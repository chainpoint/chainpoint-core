# chainpoint-lib

## Proof State Models

Proof state models allow the service to write to persistent storage. All adapters conform to a common interface, allowing easy code-based or programmatic switching between different storage providers.

The following is a description of methods that must be defined in a storage adapter: 

| Name           | Description  | Returns  |
| :------------- |:-------------|:-------------|
| getHashIdsByAggIdAsync(aggId)     | gets all hash ids associated with an aggregation event | result array containing hash id objects |
| getHashIdsByBtcTxIdAsync(btcTxId)     | gets all hash ids associated with a btcTxId | result array containing hash id objects |
| getAggStateObjectByHashIdAsync(hashId)     | gets the agg state object for a given hash id | an agg state object |
| getCalStateObjectByAggIdAsync(aggId)     | gets the cal state object for a given agg id | a cal state object |
| getAnchorBTCAggStateObjectByCalIdAsync(calId)     | gets the anchor agg state object for a given cal id | an anchor agg state object |
| getBTCTxStateObjectByAnchorBTCAggIdAsync(anchorBTCAggId)     | gets the btctx state object for a given anchor agg id | a btctx state object |
| getBTCHeadStateObjectByBTCTxIdAsync(btcTxId)     | gets the btchead state object for a given btctx id | a btchead state object |
| getAggStateObjectsByAggIdAsync(aggId)     | gets all agg state data for a given agg id | result array containing agg state objects |
| getCalStateObjectsByCalIdAsync(calId)     | gets all cal state data for a given cal id | result array containing cal state objects |
| getAnchorBTCAggStateObjectsByAnchorBTCAggIdAsync(anchorBTCAggId)     | gets all anchor agg state data for a given anchor agg id | result array containing anchor agg state objects |
| getBTCTxStateObjectsByBTCTxIdAsync(btcTxId)     | gets all btctx state data for a given btctx id | result array containing btctx state objects |
| getBTCHeadStateObjectsByBTCHeadIdAsync(btcHeadId)     | gets all btchead state data for a given btchead id | result array containing btchead state objects |
| writeAggStateObjectAsync(stateObject)     | write the agg state object to storage | boolean indicating success |
| writeAggStateObjectsAsync(stateObjects)     | write multiple agg state objects to storage individually within a loop| boolean indicating success |
| writeAggStateObjectsBulkAsync(stateObjects, transaction)     | write multiple agg state objects to storage in one bulk insert | boolean indicating success |
| writeBTCTxStateObjectAsync(stateObject)     | write the btctx state object to storage | boolean indicating success |
| writeBTCHeadStateObjectAsync(stateObject)     | write the btchead state object to storage | boolean indicating success |
| deleteProcessedHashesFromAggStatesAsync()     | prune records from agg\_states table | integer |
| deleteCalStatesWithNoRemainingAggStatesAsync()     | prune records from cal\_states table | integer |
| deleteAnchorBTCAggStatesWithNoRemainingCalStatesAsync()     | prune records from anchor\_agg\_states table | integer |
| deleteBtcTxStatesWithNoRemainingAnchorBTCAggStatesAsync()     | prune records from btctx\_states table | integer |
| deleteBtcHeadStatesWithNoRemainingBtcTxStatesAsync()     | prune records from btchead\_states table | integer |


## Proof State Models Configuration
Configuration parameters will be stored in environment variables. Environment variables can be overridden through the use of a .env file. 

The following are the descriptions of the configuration parameters:

| Name           | Description  | Default |
| :------------- |:-------------|:--------|
| COCKROACH_HOST | CockroachDB host or IP | 'roach1' |
| COCKROACH_PORT | CockroachDB port | 26257 |
| COCKROACH_DB_NAME | CockroachDB name | 'chainpoint' |
| COCKROACH_DB_USER | CockroachDB user | 'chainpoint' |
| COCKROACH_DB_PASS | CockroachDB password | '' |
| COCKROACH_TLS_CA_CRT | CockroachDB TLS CA Cert | '' |
| COCKROACH_TLS_CLIENT_KEY | CockroachDB TLS Client Key | '' |
| COCKROACH_TLS_CLIENT_CRT | CockroachDB TLS Client Cert | '' |

## Proof State Models Schema

### agg\_states
| Column         | Type         | Description  | Indexed |
| :------------- |:-------------|:-------------|:--------|
| hash\_id        | UUID         | the submitted hash's unique identifier | primary key |
| hash            | String       | the submitted hash value  |   |
| agg\_id         | UUID         | the aggregation event unique identifier | y |
| agg\_state      | Text         | the chainpoint operations connecting the hash to the aggregation event's root value |   |

### cal\_states
| Column          | Type         | Description  | Indexed |
| :-------------  |:-------------|:-------------|:--------|
| agg\_id         | UUID         | the aggregation event unique identifier | primary key |
| cal\_id         | UUID         | the calendar aggregation event unique identifier | y |
| cal\_state      | Text         | the chainpoint operations connecting the aggregation event's root value to a calendar anchor |   |

### anchor\_agg\_states
| Column          | Type         | Description  | Indexed |
| :-------------  |:-------------|:-------------|:--------|
| cal\_id         | UUID         | the calendar aggregation event unique identifier | primary key |
| anchor\_agg\_id         | UUID         | the anchor aggregation event unique identifier | y |
| anchor\_agg\_state      | Text         | the chainpoint operations connecting the calendar block hash value to the anchor aggregation event's root value |   |

### btctx\_states
| Column          | Type         | Description  | Indexed |
| :-------------  |:-------------|:-------------|:--------|
| anchor\_agg\_id | UUID         | the anchor aggregation event unique identifier | primary key |
| btctx\_id         | STRING         | the bitcoin transaction id value | y |
| btctx\_state      | Text         | the chainpoint operations connecting the anchor aggregation root value to the bitcoin transaction body value |   |

### btchead\_states
| Column          | Type         | Description  | Indexed |
| :-------------  |:-------------|:-------------|:--------|
| btctx\_id | String         | the bitcoin transaction id value | primary key |
| btchead\_height         | Integer         | the bitcoin block height for the block cointaining the transaction | y |
| btchead\_state      | Text         | the chainpoint operations connecting the bitcoin transaction body value to a btc anchor |   |

### btchead\_states
| Column          | Type         | Description  | Indexed |
| :-------------  |:-------------|:-------------|:--------|
| btctx\_id | String         | the bitcoin transaction id value | primary key |
| btchead\_height         | Integer         | the bitcoin block height for the block cointaining the transaction | y |
| btchead\_state      | Text         | the chainpoint operations connecting the bitcoin transaction body value to a btc anchor |   |
