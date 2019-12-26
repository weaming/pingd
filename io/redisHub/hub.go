package redisHub

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

const (
	API_URL = "https://hub.drink.cafe/http"

	TYPE_PLAIN    = "PLAIN"
	TYPE_MARKDOWN = "MARKDOWN"
	TYPE_JSON     = "JSON"
	TYPE_HTML     = "HTML"
	TYPE_IMAGE    = "IMAGE"

	ACTION_PUB = "PUB"
	ACTION_SUB = "SUB"
)

type PubMessage struct {
	Action  string          `json:"action"`
	Topics  []string        `json:"topics"`
	Message *PayloadMessage `json:"message"`
}

type PayloadMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func NewPubMessage(msgType, msgData string, topics []string) *PubMessage {
	data := &PayloadMessage{msgType, msgData}
	return &PubMessage{ACTION_PUB, topics, data}
}

func PostJson(api string, data interface{}) (map[string]interface{}, error) {
	bytesRepresentation, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(api, "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func PostToHub(data *PubMessage) error {
	log.Printf("post to topics %v\n", data.Topics)
	resp, err := PostJson(API_URL, data)
	log.Println(resp)
	return err
}
