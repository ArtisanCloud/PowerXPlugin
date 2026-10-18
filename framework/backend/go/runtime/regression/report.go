package regression

import (
	"encoding/xml"
	"time"
)

func Export(d RunDetail, events []RegressionEvent) Report {
	return Report{SchemaVersion: "v1", Run: d.Run, CaseExecutions: d.CaseExecutions, Events: events, ExportedAt: time.Now().UTC()}
}

type junitCase struct {
	Name    string       `xml:"name,attr"`
	Failure *junitReason `xml:"failure,omitempty"`
	Skipped *junitReason `xml:"skipped,omitempty"`
}
type junitReason struct {
	Message string `xml:"message,attr"`
}
type junitSuite struct {
	XMLName xml.Name    `xml:"testsuite"`
	Name    string      `xml:"name,attr"`
	Cases   []junitCase `xml:"testcase"`
}

func JUnit(d RunDetail) ([]byte, error) {
	s := junitSuite{Name: "regression", Cases: []junitCase{}}
	for _, v := range d.CaseExecutions {
		c := junitCase{Name: v.DefinitionSnapshot.Key}
		if v.Verdict == "failed" {
			c.Failure = &junitReason{Message: v.ReasonCode}
		} else if v.Verdict != "passed" {
			c.Skipped = &junitReason{Message: v.ReasonCode}
		}
		s.Cases = append(s.Cases, c)
	}
	return xml.MarshalIndent(s, "", "  ")
}
