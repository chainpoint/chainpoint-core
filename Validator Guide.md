## Validator Guide

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
    1. This consists of sending a transaction with the format `val:<ID>!<b64_public_key>!<voting_power>` to the network, where voting power is the whole integer representing how many votes the validator should have each block election round. 
6. An existing validator whitelists the new node operator’s IP, opening up ports 26656 and 26657 to the new validator. 
7. The new validator begins operating their node with the `make up` command. It will fastsync and begin voting on new blocks. 

### Change Required Lightning Stake

1. Upon setting the `update_stake` config value, a core will automatically submit a `CHNGSTK` tx to tendermint consensus.
2. Each other core will only confirm `CHNGSTK` if the value is equal to their existing `stake_per_core` configuration value
    1. In practice this means that 2/3 of all validators must share this configuration value. 
    2. If the value changes then these Cores must restart in order to get the new config value, so they can approve the `CHNGSTK` tx
3. Upon initializing, Cores will automatically read the latest `CHNGSTK` value from the tendermint index and use this value for the staking requirement 