# Operator's Guide

## Setup

The Makefile process described in the [quickstart](https://github.com/chainpoint/chainpoint-core#quick-start) will install and configure golang on the target box, as well as build the chainpoint-core binary. 
After compilation, there are two possible modes:

#### Public Chainpoint Network
 1. Manually running the binary will start the setup wizard. Be sure to select the "mainnet" and "Public chainpoint network" options, then put in your server's public IP. Contact Tierion to allow your IP through the seed node's firewall.
 2. The in-built Lightning Node will then initialize and print a bitcoin address, as well as the amount of bitcoin necessary to stake with the public Chainpoint Network. Fund this address with the requested amount. 
 3. At this point you can kill the process and daemonize the node for long-term running, if you wish. The commands for this are `make install-daemon` and `make start-daemon`. Running `make log-daemon` will let you see the logs. 
 4. After a few confirmations, your Core will be able to send a JWK message declaring its public key, at which point the stake will be checked and verified by the Validators. 
 5. If you wish, and after a certain amount of successful time on the Network, our nodes can collectively vote to elevate your Core to Validator status. 

#### Private Chainpoint Network
 1. Manually running the binary will start the setup wizard. Be sure to select the "mainnet" and "Standalone Mode" options, then put in your server's public IP.
 2. The in-built Lightning Node will then initialize and print a bitcoin address. You'll need to fund this address with funds sufficient to fund bitcoin anchoring (segwit OP_RETURN transactions). 
 3. If you're joining another private node to create a private Chainpoint network, specify `"SEEDS=<seed_node_id@<seed_node_IP>:26656"` in the config file (by default at `~/.chainpoint/core/core.conf`). The seed node id can be found from retrieving the `id` json field at `http://<seed_node_IP>/status`.
 4. At this point you can kill the process and daemonize the node for long-term running, if you wish. The commands for this are `make install-daemon` and `make start-daemon`. Running `make log-daemon` will let you see the logs. 

## Usage

### With Chainpoint Gateways
In high-usage or public-facing situations, it is recommended to use [Chainpoint Gateway](https://github.com/chainpoint/chainpoint-gateway) as a public-facing service in front of your Core. 
It will need to open a channel to your Core in order to submit hashes, or use `AGGREGATOR_WHITELIST=<gateway_ips>` (Core) and `NO_LSAT_CORE_WHITELIST=<core_ips>` (Gateway) to skip lightning usage.

### Directly With Core 
However, it is possible to use Core directly as a proof generator. Configure your Core to accept requests from your client IPs by adding 
```
AGGREGATOR_WHITELIST=<client_ips>
REMOVE_RATE_LIMITS=true
```
to your config file (by default at `~/.chainpoint/core/core.conf`).

Usage of Core's API is located in [Usage.md](Usage.md)

## Updates

Core uses semantic versioning:
- Non-consensus-related updates and critical fixes will result in a patch (point release, ie 1.3.0 => 1.3.1)
- New features will result in a minor version release (ie 1.3.1 => 1.4.0)
- Any changes to the consensus protocol will result in a major version release (ie 1.4.0 => 2.0.0) 

Any updates to the consensus protocol will require the cooperation of all Core operators. 
Notice of an available update will be given through email for regular Core operators, whereas Validator operators will coordinate in real time over Discord. 

## Validators

Chainpoint Validators have the final responsibility for forming blocks and validating transactions.
They also must vote to add additional validators and increase/decrease the lightning stake required to join the network

### Adding Validators

1. New person or entity applies to be a validator node. 
2. Provision a server, clone the Chainpoint-Core repo, and run `make install-deps` and `make build`
3. Upon running Core for the first time, a setup wizard will do the following:
    1. Generate the node's public key and ID
    2. Ingest the new node’s IP
4. Upon 2nd start (normal operation), the node will print something like
    ```
   Core ID set                                  ID=AF12ACE1A4058F4E60723930E96200EB605D0B36
   Core Tendermint Publickey set                Key=+MU67U5bacm7H/2ZWaAltvchl7RyXwHJ8pl6lIq7zYw=
   ```
   These must be given to the validator agreeing to run the next steps.
5. An existing validator operator aggrees to broadcast a VAL transaction at a particular block height agreed upon by the rest of the Validators. 
    1. This consists of starting all validators with the argument `proposed_val=val:<ID>!<b64_public_key>!<voting_power>!<block_height>` to the network, where
        - ID is the Tendermint ID of the node, shown above 
        - PubKey is the base 64-encoded tendermint public key of the node
        - Voting power is the amount of weight the node has while voting on new blocks (should be 1 in nearly all cases)
        - Block height is the height the new validator will take effect at. 
        
6. An existing validator whitelists the new node operator’s IP, opening up ports 26656 and 26657 to the new validator. 
7. The new validator begins operating their node with the `make up` command. It will fastsync and begin voting on new blocks. 

### Change Required Lightning Stake

1. Upon setting the `update_stake` config value, a core will automatically submit a `CHNGSTK` tx to tendermint consensus.
2. Each other core will only confirm `CHNGSTK` if the value is equal to their existing `stake_per_core` configuration value
    1. In practice this means that 2/3 of all validators must share this configuration value. 
    2. If the value changes then these Cores must restart in order to get the new config value, so they can approve the `CHNGSTK` tx
3. Upon initializing, Cores will automatically read the latest `CHNGSTK` value from the tendermint index and use this value for the staking requirement 

## Support

Email:    `team@tierion.com`, `ops@tierion.com`

Report Bugs on our Github: https://github.com/chainpoint/chainpoint-core/issues

We also have a discord. Please email us for an invite. 