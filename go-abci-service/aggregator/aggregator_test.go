package aggregator

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/streadway/amqp"
)

func TestHashResult(t *testing.T) {
	item := types.HashItem{
		"6d627180-1883-11e7-a8f9-edb8c212ef23",
		"ed10960ccc613e4ad0533a813e2027924afd051f5065bb5379a80337c69afcb4",
	}
	item2 := types.HashItem{
		"6d627180-1883-11e7-a8f9-edb8c212ef23",
		"aa10960ccc613e4ad0533a813e2027924afd051f5065bb5379a80337c69afcb4",
	}
	itemBytes, _ := json.Marshal(item)
	itemBytes2, _ := json.Marshal(item2)
	msgArray := []amqp.Delivery{
		amqp.Delivery{
			nil,
			amqp.Table{},
			"",
			"",
			0,
			0,
			"",
			"",
			"",
			"",
			time.Now(),
			"",
			"",
			"",
			"",
			0,
			0,
			false,
			"",
			"",
			itemBytes,
		},
		amqp.Delivery{
			nil,
			amqp.Table{},
			"",
			"",
			0,
			0,
			"",
			"",
			"",
			"",
			time.Now(),
			"",
			"",
			"",
			"",
			0,
			0,
			false,
			"",
			"",
			itemBytes2,
		},
	}
	agg := ProcessAggregation("", msgArray, "")
	if agg.AggRoot != "58f42246b9c6d303e33206d461e05f3e2292d8eddfce92b7434f1d8be9f0e2c1" {
		t.Errorf("merkle root value should be 58f42246b9c6d303e33206d461e05f3e2292d8eddfce92b7434f1d8be9f0e2c1, got: %s", agg.AggRoot)
	}
}
