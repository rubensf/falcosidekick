// SPDX-License-Identifier: MIT OR Apache-2.0

package outputs

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	nats "github.com/nats-io/nats.go"

	"github.com/falcosecurity/falcosidekick/types"
)

var slugRegExp = regexp.MustCompile("[^a-z0-9]+")

const defaultNatsSubjects = "falco.<priority>.<rule>"

// NatsPublish publishes event to NATS
func (c *Client) NatsPublish(falcopayload types.FalcoPayload) {
	c.Stats.Nats.Add(Total, 1)

	subject := c.Config.Nats.SubjectTemplate
	if len(subject) == 0 {
		subject = defaultNatsSubjects
	}

	subject = strings.ReplaceAll(subject, "<priority>", strings.ToLower(falcopayload.Priority.String()))
	subject = strings.ReplaceAll(subject, "<rule>", strings.Trim(slugRegExp.ReplaceAllString(strings.ToLower(falcopayload.Rule), "_"), "_"))

	nc, err := nats.Connect(c.EndpointURL.String())
	if err != nil {
		c.setNatsErrorMetrics()
		log.Printf("[ERROR] : NATS - %v\n", err)
		return
	}
	defer nc.Flush()
	defer nc.Close()

	j, err := json.Marshal(falcopayload)
	if err != nil {
		c.setStanErrorMetrics()
		log.Printf("[ERROR] : STAN - %v\n", err.Error())
		return
	}

	err = nc.Publish(subject, j)
	if err != nil {
		c.setNatsErrorMetrics()
		log.Printf("[ERROR] : NATS - %v\n", err)
		return
	}

	go c.CountMetric("outputs", 1, []string{"output:nats", "status:ok"})
	c.Stats.Nats.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "nats", "status": OK}).Inc()
	c.OTLPMetrics.Outputs.With(attribute.String("destination", "nats"), attribute.String("status", OK)).Inc()
	log.Printf("[INFO]  : NATS - Publish OK\n")
}

// setNatsErrorMetrics set the error stats
func (c *Client) setNatsErrorMetrics() {
	go c.CountMetric(Outputs, 1, []string{"output:nats", "status:error"})
	c.Stats.Nats.Add(Error, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "nats", "status": Error}).Inc()
	c.OTLPMetrics.Outputs.With(attribute.String("destination", "nats"),
		attribute.String("status", Error)).Inc()

}
