const net = require('net');
const HyperDHT = require('hyperdht');

async function main() {
    const connString = process.argv[2];

    if (!connString) {
        console.error('Usage: node entry.js hs://<key>');
        process.exit(1);
    }

    const key = connString.replace('hs://', '');
    const dht = new HyperDHT();

    // Determine Pipe Transport
    let pipePath = null;
    if (process.env.GS_NAMED_PIPE) {
        pipePath = process.env.GS_NAMED_PIPE;
        console.log(`[JS] Using Named Pipe: ${pipePath}`);
    } else if (process.env.GS_PIPE_FD) {
        pipePath = { fd: parseInt(process.env.GS_PIPE_FD) };
        console.log(`[JS] Using Socketpair (FD: ${process.env.GS_PIPE_FD})`);
    }

    if (!pipePath) {
        console.error('[JS] Fatal: No pipe transport established.');
        process.exit(1);
    }

    console.log(`[JS] Connecting to DHT Key: ${key}`);

    const server = net.createServer((bridgeSocket) => {
        console.log('[JS] New Connection from P2P Network');

        const clientSocket = net.connect(pipePath, () => {
            bridgeSocket.pipe(clientSocket).pipe(bridgeSocket);
        });

        clientSocket.on('error', (err) => {
            console.error('[JS] Bridge Connection Error:', err.message);
            bridgeSocket.destroy();
        });

        bridgeSocket.on('error', (err) => {
            console.error('[JS] Client Socket Error:', err.message);
            clientSocket.destroy();
        });
    });

    const node = dht.createServer((bridgeSocket) => {
        const clientSocket = net.connect(pipePath, () => {
            bridgeSocket.pipe(clientSocket).pipe(bridgeSocket);
        });
    });

    const keyPair = HyperDHT.keyPair(Buffer.from(key, 'hex'));
    await node.listen(keyPair);
    console.log('[JS] P2P Node Listening on DHT Key');
}

main().catch(console.error);
