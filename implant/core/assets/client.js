const net = require('net');
const HyperDHT = require('hyperdht');
const { program } = require('commander');

program
  .option('-p, --port <number>', 'Sliver mTLS port', '8888')
  .parse(process.argv);

const options = program.opts();
const dht = new HyperDHT();

async function startBridge() {
  const keyPair = dht.defaultKeyPair;
  const publicKey = keyPair.publicKey.toString('hex');

  console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');
  console.log('\x1b[1m\x1b[32m  GHOSTSHIP BRIDGE (v1.0.0) - OPERATOR SIDE\x1b[0m');
  console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');
  console.log(`\x1b[1mSliver Port:\x1b[0m      ${options.port}`);
  console.log(`\x1b[1mConnection Key:\x1b[0m   hs://${publicKey}`);
  console.log('\x1b[1m\x1b[32m======================================================================\x1b[0m');

  const server = dht.createServer((bridgeSocket) => {
    console.log('[*] New P2P Connection established');

    const sliverSocket = net.connect(options.port, '127.0.0.1', () => {
      bridgeSocket.pipe(sliverSocket).pipe(bridgeSocket);
    });

    sliverSocket.on('error', (err) => {
      console.error('[!] Sliver connection error:', err.message);
      bridgeSocket.destroy();
    });

    bridgeSocket.on('error', (err) => {
      console.error('[!] Bridge socket error:', err.message);
      sliverSocket.destroy();
    });
  });

  await server.listen(keyPair);
}

startBridge().catch(console.error);
