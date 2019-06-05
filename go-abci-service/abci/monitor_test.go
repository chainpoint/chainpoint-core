package abci

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/stretchr/testify/assert"
)

func TestMonitorConsumeBtcTxMsg(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	app := DeclareABCI()
	tx := types.Tx{"BTC-M", "{\"anchor_btc_agg_id\":\"cb8196f7-6d6f-b0a2-b542-9355f762acb3\"," +
		"\"anchor_btc_agg_root\":\"cb8196f76d6fb0a2b5429355f762acb32c919402575135315d418875255929f5\"," +
		"\"proofData\":[{\"cal_id\":\"ab79e05dec291d1337e87719ee7493630590b26d132d39f21bf151f0be254f6f\"," +
		"\"proof\":[{\"r\":\"90e52b2373ad804677a3e3b9a5d9de7ebef6858158b6387d9e34bc5ad1603e70\"},{\"op\":\"sha-256\"}," +
		"{\"r\":\"0e664e9dea8466a8a887d68e0da428c27da0bc28fd47f575174726534cb8cb30\"},{\"op\":\"sha-256\"}," +
		"{\"r\":\"6c94a3c5ab006b9ad6d4533043b139bd5999bf8d4d61998221324dabf2472add\"},{\"op\":\"sha-256\"}," +
		"{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"}," +
		"{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"},{\"op\":\"sha-256\"}]}," +
		"{\"cal_id\":\"6e66c75ac11cd1ccf4c470a029b66e7c2775d5939e6464c422cbb91374d7a612\"," +
		"\"proof\":[{\"l\":\"5e0eb05caea431b1a79ac0e454fe0da0dd6cac5ace86addc7ecd3fdddf2e3d95\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"0e664e9dea8466a8a887d68e0da428c27da0bc28fd47f575174726534cb8cb30\"},{\"op\":\"sha-256\"},{\"r\":\"6c94a3c5ab006b9ad6d4533043b139bd5999bf8d4d61998221324dabf2472add\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"463735eb844f79824d8fb62c9f6e0a5f49d32b1400398e054e579293d0538eef\",\"proof\":[{\"r\":\"7f84963c0a23f1048749f6d2714c9238db863a28cbfd92be0aa4849c075defb7\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"824a9fef3212f29c096ebc6561d31de12e3bef70121a9bc494df0025b0138cef\"},{\"op\":\"sha-256\"},{\"r\":\"6c94a3c5ab006b9ad6d4533043b139bd5999bf8d4d61998221324dabf2472add\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"36c30257916e2794aee29a80974a7c5cac2c952edf169ab18dd805d9b9c79709\",\"proof\":[{\"l\":\"a0a202c92f92d779d2d3fbe1e96c64e818e208222bda190c28041203eb98a1f0\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"824a9fef3212f29c096ebc6561d31de12e3bef70121a9bc494df0025b0138cef\"},{\"op\":\"sha-256\"},{\"r\":\"6c94a3c5ab006b9ad6d4533043b139bd5999bf8d4d61998221324dabf2472add\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"f71a7c9e0d7f0a6214d9ae9ef3812a27cf141117f86760a1980dbc713c86274c\",\"proof\":[{\"r\":\"00df33f02a2408f61dd2999f0e329f2527f51684e93aff486731eea802107572\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"7fb2f479e7cbc2a24ed7e2a4822edb345ecedb739123fbc9b2f415c8f7b45ccc\"},{\"op\":\"sha-256\"},{\"l\":\"56398698afd1e7085fb544ba3b4e3d28be58956d9d5c7252889f853b7cc302c9\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"4d6da47034c76bebc16de0404d32bb0a87852d23c9be7fab9f6b1fbb0f991b16\",\"proof\":[{\"l\":\"3a8b057d637e4a9f800197cd826c2344e7b51d18121d0a7d4f774b5ef23a3b00\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"7fb2f479e7cbc2a24ed7e2a4822edb345ecedb739123fbc9b2f415c8f7b45ccc\"},{\"op\":\"sha-256\"},{\"l\":\"56398698afd1e7085fb544ba3b4e3d28be58956d9d5c7252889f853b7cc302c9\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"8587b9733f6921d887527812bc937d19f740fbc3182e9cb5936b920162470c52\",\"proof\":[{\"r\":\"dfbf5cc9fc566a555e8f8b83d1b39ed96b926a0b969215b9db6ad5355a6205a6\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"2784960fa81376202fbd3dfd2f05a318c2c5362396429386201f89bd5f96ce56\"},{\"op\":\"sha-256\"},{\"l\":\"56398698afd1e7085fb544ba3b4e3d28be58956d9d5c7252889f853b7cc302c9\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"f6c915e5e2ee83b4b3b943a5b4d713904380b3cf90043e891ed35a872bfca353\",\"proof\":[{\"l\":\"f8373d082976b69a4678ddc936cd30d81dcef9c64655f454c32c45d7e8ced2e1\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"2784960fa81376202fbd3dfd2f05a318c2c5362396429386201f89bd5f96ce56\"},{\"op\":\"sha-256\"},{\"l\":\"56398698afd1e7085fb544ba3b4e3d28be58956d9d5c7252889f853b7cc302c9\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"b4bdbc3ad4d71fe63d2b9b1ea474b32bf7305cce7a1ae9544c7762204f58e24b\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"24edb0086aadc356b1f5fce264a2c4a78461f7f78cf62fc337a22f880775db0d\",\"proof\":[{\"r\":\"a3c7aa56f9ab990d14574465809846de8d83d12d8b4145d706243d80be85515c\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"969b2a61f22d2fdf6b747b2fb7d2f0de9c33769be284d3f2738cc1db2dc7631b\"},{\"op\":\"sha-256\"},{\"r\":\"f1e025cc4a039ba8ee39827d6cfff56981e9a6dac840872e318970e5c8c99b21\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"283e467e2c4886ab1ddbd4824c1a4b8c9d4ac5adbe848d625d80d507c6883b96\",\"proof\":[{\"l\":\"61e51b8ff84ab9f50b45d51ed43d14a40dfeb2d01d251455a81fd2cc40b6d2ea\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"969b2a61f22d2fdf6b747b2fb7d2f0de9c33769be284d3f2738cc1db2dc7631b\"},{\"op\":\"sha-256\"},{\"r\":\"f1e025cc4a039ba8ee39827d6cfff56981e9a6dac840872e318970e5c8c99b21\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"00da5a5f68c1ce5d2f4b9be37ebe95a997e536684bd4c4a37a6c2c4b99c261e8\",\"proof\":[{\"r\":\"a050d7e40bb8d7097139b3ede188b77794ed4d270c6e1745c04ece8c66013632\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"65d1e2c8291018fc2950d683238ddfb350cbd21b578f0677df732fb19f061173\"},{\"op\":\"sha-256\"},{\"r\":\"f1e025cc4a039ba8ee39827d6cfff56981e9a6dac840872e318970e5c8c99b21\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"7ceb1d9c0bac376000ec866a1ad60db5240e855e2e6964ce3fc6f7af6f381030\",\"proof\":[{\"l\":\"1ac1722d23247472179f249c8b7c2e9c046931b3e251649da64f3053312a49c7\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"65d1e2c8291018fc2950d683238ddfb350cbd21b578f0677df732fb19f061173\"},{\"op\":\"sha-256\"},{\"r\":\"f1e025cc4a039ba8ee39827d6cfff56981e9a6dac840872e318970e5c8c99b21\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"2c62474649811570c351f5a15340d80b94475d524e4d9613de5c10211cb2982d\",\"proof\":[{\"r\":\"b5dc340e418eaf51fe66df8d436f1141c4825ae341e22edf5b8fa018e5f577d3\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"ca5ccd63910141b828494ec29f60a1f1faab16d7e2fa66c721158a7867be79fd\"},{\"op\":\"sha-256\"},{\"l\":\"54c89582201dd4cec8785dbba826424aac35580a8c299453d1c2533006b1f6b8\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"72221d0bf3992c8c7fbcc164758f7b24fc35ec731dfeb358a90e03647a7c9199\",\"proof\":[{\"l\":\"b3b6690bdbc52b25699f4a4caa4b690aae69d4cd18d2a4d0a836c57ede84bd43\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"ca5ccd63910141b828494ec29f60a1f1faab16d7e2fa66c721158a7867be79fd\"},{\"op\":\"sha-256\"},{\"l\":\"54c89582201dd4cec8785dbba826424aac35580a8c299453d1c2533006b1f6b8\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"642120957cf609945c3c6337177d5f15ec6a407d75776b4bcedaad23aa466657\",\"proof\":[{\"r\":\"f92ec04c307726164b9031eaf8449966484f46bf3cf13c659029969bc3db1fea\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"eeeee80b6bac3bd69c258ea53a6b0330c7f7dd9be051732cab8f4ff40c6a5519\"},{\"op\":\"sha-256\"},{\"l\":\"54c89582201dd4cec8785dbba826424aac35580a8c299453d1c2533006b1f6b8\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"bc5b91e0d42789cdd5c0ad4870bff6420e9bab29980e56edbaaf92d3018edb01\",\"proof\":[{\"l\":\"17c48a7d925dd63dccdd2af25e8fba51556a33026ec14b5366cf49546d7726b3\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"eeeee80b6bac3bd69c258ea53a6b0330c7f7dd9be051732cab8f4ff40c6a5519\"},{\"op\":\"sha-256\"},{\"l\":\"54c89582201dd4cec8785dbba826424aac35580a8c299453d1c2533006b1f6b8\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"4cd5a8d45894d29b8ea6bec261e6ca93d8fd1274cf16faafa541a6a758d56336\"},{\"op\":\"sha-256\"},{\"r\":\"39cabed0a6ecd4149674a4683974e1c5e46757073fb289e8e08a2b98d1410673\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"877e5eb96b9f7014b44fe894c1e4051d91875fc561997f992441671a6d990c64\",\"proof\":[{\"r\":\"931babb4cf4825784c5a51123e20e989ecce7f449b577b94fa5d41a14e867272\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"df82644287496c1959e1e2aa031949cf1d12ecc80a2f4c83644e912dbea3eb01\"},{\"op\":\"sha-256\"},{\"r\":\"9c4fbd46a7808508c51e5fe59e2f81b098df8b728a848472b1fe7b0e1e39011e\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"5de64fa0cca31aa8b045b9f043d5c88d788acccc5575ca29e836fe2a0fbcc8fd\"},{\"op\":\"sha-256\"}]},{\"cal_id\":\"770cd50566d21cd0f4438b4450da5f8651e75259f0adf2c1ab4edb16533a6227\"," +
		"\"proof\":[{\"l\":\"e333bca2f64ab400b32f7ef649f46f47fb8881ad1d7ba5c1c92bca18778791ca\"},{\"op\":\"sha-256\"},{\"r\":\"df82644287496c1959e1e2aa031949cf1d12ecc80a2f4c83644e912dbea3eb01\"},{\"op\":\"sha-256\"}," +
		"{\"r\":\"9c4fbd46a7808508c51e5fe59e2f81b098df8b728a848472b1fe7b0e1e39011e\"},{\"op\":\"sha-256\"},{\"l\":\"5de64fa0cca31aa8b045b9f043d5c88d788acccc5575ca29e836fe2a0fbcc8fd\"},{\"op\":\"sha-256\"}]}," +
		"{\"cal_id\":\"afee7b847ad1916e94987dac01f274e9f70c5875e5a8b4e820b29501d1e45db7\",\"proof\":[{\"r\":\"f9fbd209e68150e7f3b80e740e3929eb3b470f7634b9749f3d0f5b3fa9080419\"},{\"op\":\"sha-256\"}," +
		"{\"l\":\"013783fa8f3efa14f2312d3f91eca939301d9d31b4584eb015658d3ec925d56d\"},{\"op\":\"sha-256\"},{\"r\":\"9c4fbd46a7808508c51e5fe59e2f81b098df8b728a848472b1fe7b0e1e39011e\"},{\"op\":\"sha-256\"}," +
		"{\"l\":\"5de64fa0cca31aa8b045b9f043d5c88d788acccc5575ca29e836fe2a0fbcc8fd\"},{\"op\":\"sha-256\"}]},{\"cal_id\":\"16e37c35eb12b02b40513234e1fff780011c806f3ad951c6485cba5e186c10ac\"," +
		"\"proof\":[{\"l\":\"4315c8099e6652452ef3f9cf048eb75ee5ba523e0599df8195133c7a42631ffa\"},{\"op\":\"sha-256\"},{\"l\":\"013783fa8f3efa14f2312d3f91eca939301d9d31b4584eb015658d3ec925d56d\"}," +
		"{\"op\":\"sha-256\"},{\"r\":\"9c4fbd46a7808508c51e5fe59e2f81b098df8b728a848472b1fe7b0e1e39011e\"},{\"op\":\"sha-256\"},{\"l\":\"5de64fa0cca31aa8b045b9f043d5c88d788acccc5575ca29e836fe2a0fbcc8fd\"}," +
		"{\"op\":\"sha-256\"}]},{\"cal_id\":\"990a3f1a723058a9904bb88948653295aba72d6269ec6bad8c0fff7ce32ff4e0\",\"proof\":[{\"l\":\"10c3a1242f77069f19f324071b3d328c3e1ad6c30a5d5549824c186573487acc\"}," +
		"{\"op\":\"sha-256\"},{\"l\":\"5de64fa0cca31aa8b045b9f043d5c88d788acccc5575ca29e836fe2a0fbcc8fd\"},{\"op\":\"sha-256\"}]}],\"btctx_id\":\"c1d117a7bad78af98ce4fba7243789b5fca1f084f6191f181e9bba0c102d2e7f\"," +
		"\"btctx_body\":\"0100000001b7ac979f1eab85f2f4c8a743b0699b97c9df15b60db4c782bd68d0145570a203010000008a47304402207f3aec63559c7905f7dbd665de01413161b71da289dd7e9d2abad5d51e4f8f0c02201e9b015979629aa2d565a0d890206620aebee06e0e16ccc0261055917ed18d38014104494fb4afff6796542983a3aa703558a654d663d15d65595dac07af1e97e32d7c8ff39f9a1ed3b1b441d141266018d4b3ca3d2d620024290220f323a341373fe8ffffffff020000000000000000226a20cb8196f76d6fb0a2b5429355f762acb32c919402575135315d418875255929f5b6d62e03000000001976a914c83ef0e339562a6ce3b6a42593f20b63f74a4b1188ac00000000\"}",
		2,
		1553112582,
		"me_id",
		"signature",
	}
	err := app.ConsumeBtcTxMsg([]byte(tx.Data))
	assert.Equal(err, nil, "err from ConsumeBtcTxMsg should be nil")

	// Test for msg presence in proofstate queueblah
	session, err := rabbitmq.ConnectAndConsume(rabbitTestURI, "work.proofstate")
	assert.Equal(nil, err, "error from rabbitmq dial should be nil")
	for m := range session.Msgs {
		assert.Equal(m.Type, "btctx", "rabbitmq message type should be 'btctx'")
		var stateObj types.BtcTxProofState
		err = json.Unmarshal([]byte(tx.Data), &stateObj)
		assert.Equal(err, nil, "err upon unmarshalling BTC-M data should be nil")
		assert.Equal(stateObj.BtcTxID, "c1d117a7bad78af98ce4fba7243789b5fca1f084f6191f181e9bba0c102d2e7f", "BTC Tx ID should match between monitor output and BTC-M message")
		m.Ack(false)
		break
	}
	err = session.End()

	// Test for msg presence in btcmon queue
	session, err = rabbitmq.ConnectAndConsume(rabbitTestURI, "work.btcmon")
	for m := range session.Msgs {
		assert.Equal(m.Type, "", "rabbitmq message type should be empty string")
		var stateObj types.BtcTxProofState
		err = json.Unmarshal([]byte(tx.Data), &stateObj)
		assert.Equal(err, nil, "err upon unmarshalling BTC-M data should be nil")
		assert.Equal(stateObj.BtcTxID, "c1d117a7bad78af98ce4fba7243789b5fca1f084f6191f181e9bba0c102d2e7f", "BTC Tx ID should match between monitor output and BTC-M message")
		m.Ack(false)
		break
	}
	err = session.End()
}

/*func TestMonitorConsumeBtcMonMsg(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	app := DeclareABCI()
	msg := types.BtcMonMsg{
		BtcTxID:       "blah",
		BtcHeadHeight: 1,
		BtcHeadRoot:   "blahblah",
	}
	msgBytes, err := json.Marshal(msg)
	assert.Equal(err, nil, "err from marshal should be nil")
	msgAmqp := amqp.Delivery{
		Acknowledger:    nil,
		Headers:         amqp.Table{},
		ContentType:     "",
		ContentEncoding: "",
		DeliveryMode:    0,
		Priority:        0,
		CorrelationId:   "",
		ReplyTo:         "",
		Expiration:      "",
		MessageId:       "",
		Timestamp:       time.Now(),
		Type:            "",
		UserId:          "",
		AppId:           "",
		ConsumerTag:     "",
		MessageCount:    0,
		DeliveryTag:     0,
		Redelivered:     false,
		Exchange:        "",
		RoutingKey:      "",
		Body:            msgBytes,
	}
	err = app.ConsumeBtcMonMsg(msgAmqp)
	assert.Equal(err, nil, "err from ConsumeBtcMonMsg should be nil")

	session, err := rabbitmq.ConnectAndConsume(rabbitTestURI, "work.proofstate")
	for m := range session.Msgs {
		assert.Equal(m.Type, "btcmon", "rabbitmq message type should be btcmon")
		m.Ack(false)
		break
	}
	err = session.End()
}*/
