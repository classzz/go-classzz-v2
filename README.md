## go-classzz-v2

The official version of the Te Waka cross-chain protocol was launched. At that time, users can use CzzSwap to exchange assets on ETH, HECO, and BSC at will. The CZZ mainnet currency will also appear in the form of ECZZ, HCZZ, and BCZZ on ETH, HECO, and BSC, and will continue to land on these ecological Dex.

Developers can also use the open source API provided by the Class ZZ community to embed the Te Waka protocol in their applications. Class ZZ (CZZ) is the world's first public chain to realize decentralized cross-chain transactions. It realizes cross-chain transactions through the native token cross-chain protocol (ie Te Waka), and is the "cross-border ship" of the blockchain world.

The Te Waka protocol is completely open source and decentralized, enabling Token to switch arbitrarily on the mainnet supported by the protocol. At present, the Te Waka protocol has successfully supported cross-chain transactions of assets on the ETH, HECO, and BSC chains, and will continue to support cross-chain transactions of public chains such as Polkadot, Solana.

<a href="https://github.com/classzz/go-classzz-v2/blob/master/COPYING"><img src="https://img.shields.io/badge/license-GPL%20%20Czzchain-lightgrey.svg"></a>

## Building the source

Building gczz requires both a Go (version 1.14 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make gczz

or, to build the full suite of utilities:

    make all

The execuable command gczz will be found in the `cmd` directory.

## Running gczz

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/classzz/go-classzz-v2/wiki/Command-Line-Options)), 
also you can quickly run your own gczz instance with a few common parameter combos.

### Running on the Classzz main network

```
$ gczz console
```

This command will:

 * Start gczz with network ID `61` in full node mode(default, can be changed with the `--syncmode` flag after version 1.1).
 * Start up Gczz's built-in interactive console,
   (via the trailing `console` subcommand) through which you can invoke all official [`web3` methods](https://github.com/classzz/go-classzz-v2/wiki/RPC-API)
   as well as Geth's own [management APIs](https://github.com/classzz/go-classzz-v2/wiki/Management-API).
   This too is optional and if you leave it out you can always attach to an already running Gczz instance
   with `gczz attach`.


### Running on the Classzz test network

To test your contracts, you can join the test network with your node.

```
$ gczz --testnet console
```

The `console` subcommand has the exact same meaning as above and they are equally useful on the
testnet too. Please see above for their explanations if you've skipped here.

Specifying the `--testnet` flag, however, will reconfigure your Gczz instance a bit:

 * Test network uses different network ID `62`
 * Instead of connecting the main Classzz network, the client will connect to the test network, which uses testnet P2P bootnodes,  and genesis states.


### Configuration

As an alternative to passing the numerous flags to the `gczz` binary, you can also pass a configuration file via:

```
$ gczz --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to export your existing configuration:

```
$ gczz --your-favourite-flags dumpconfig
```


### Running on the Classzz singlenode(private) network

To start a g
instance for single node,  run it with these flags:

```
$ gczz --singlenode  console
```

Specifying the `--singlenode` flag, however, will reconfigure your Geth instance a bit:

 * singlenode network uses different network ID `63`
 * Instead of connecting the main or test Classzz network, the client has no peers, and generate shard block without committee.

Which will start sending transactions periodly to this node and mining fruits and pow blocks.
