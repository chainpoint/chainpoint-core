# Lightning Recovery Procedure

### Problem

If a lightning wallet is failing to send transactions or open channels, it’s likely there’s one of three things wrong:


1. Incorrect commands to start LND
2. LND wallet database is corrupt
3. LND node has no valid peers

### Remedies

1. Incorrect commands to start LND
    - Use this script to start LND

2. LND wallet database is corrupt
    - Use dropwtxmgr on the wallet.db file in your lnd directory
    - Restart LND

3. LND node has no valid peers
    - Stop LND
    - Remove the peers.json file
    - Add bitcoin nodes that serve compact filters as peers from bitnodes.io
    - Start LND