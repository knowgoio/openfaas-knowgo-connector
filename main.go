// Copyright (c) Adaptant Solutions AG 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/adaptant-labs/connector-sdk/types"
	"github.com/knowgoio/knowgo-pubsub/api"
	"github.com/openfaas/faas-provider/auth"
)

func main() {
	var gatewayUsername, gatewayPassword, gatewayServer string

	flag.StringVar(&gatewayUsername, "gw-username", "admin", "Username for the OpenFaaS Gateway")
	flag.StringVar(&gatewayPassword, "gw-password", "", "Password for the OpenFaaS Gateway")
	flag.StringVar(&gatewayServer, "gateway", "", "OpenFaaS Gateway Address")

	topic := flag.String("topic", "country-change", "The topic name to/from which to publish/subscribe")
	broker := flag.String("broker", "http://localhost:8080", "The broker URI. ex: http://10.10.1.1:8080")
	apiKey := flag.String("api-key", "", "KnowGo Platform API Key")

	flag.Parse()

	var creds *auth.BasicAuthCredentials

	if gatewayPassword != "" {
		creds = &auth.BasicAuthCredentials{
			User:     gatewayUsername,
			Password: gatewayPassword,
		}
	} else {
		creds = types.GetCredentials()
	}

	var gatewayURL string

	if gatewayServer != "" {
		gatewayURL = gatewayServer
	} else {
		gatewayURL = os.Getenv("OPENFAAS_URL")
	}

	if gatewayURL == "" {
		log.Fatal("Unable to determine OpenFaaS Gateway address. Set OPENFAAS_URL or specify the -gateway flag.")
		return
	}

	config := &types.ControllerConfig{
		RebuildInterval:   time.Millisecond * 1000,
		GatewayURL:        gatewayURL,
		PrintResponse:     true,
		PrintResponseBody: true,
	}

	log.Printf("Topic: %s\tBroker: %s\n", *topic, *broker)

	controller := types.NewController(creds, config)

	receiver := ResponseReceiver{}
	controller.Subscribe(&receiver)

	controller.BeginMapBuilder()

	client := api.DefaultClientConfig()

	if *apiKey != "" {
		client.APIKey = *apiKey
	}

	if *broker != "" {
		u, err := url.Parse(*broker)
		if err == nil {
			client.Host = u.Hostname()
			client.Port, _ = strconv.Atoi(u.Port())
		}
	}

	sub, err := client.Subscribe(&api.SubscriptionRequest{
		Event: *topic,
	})
	if err != nil {
		fmt.Println("failed to subscribe to notifications: ", err)
		return
	}

	ticker := time.NewTicker(client.PollInterval).C
	for {
		select {
		case <-ticker:
			data := client.Receive(context.Background(), sub)
			if data != nil {
				log.Printf("Invoking (%s) on topic: %s, value: %s\n", gatewayURL, *topic, string(data))
				controller.InvokeWithContext(context.Background(), *topic, &data)
			}
		}
	}
}

// ResponseReceiver enables connector to receive results from the
// function invocation
type ResponseReceiver struct {
}

// Response is triggered by the controller when a message is
// received from the function invocation
func (ResponseReceiver) Response(res types.InvokerResponse) {
	if res.Error != nil {
		log.Printf("knowgo-connector got error: %s", res.Error.Error())
	} else {
		log.Printf("knowgo-connector got result: [%d] %s => %s (%d) bytes", res.Status, res.Topic, res.Function, len(*res.Body))
	}
}
