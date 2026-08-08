package main

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type engagementScope struct {
	Engagement       string           `yaml:"engagement"`
	Client           string           `yaml:"client"`
	AssessmentType   string           `yaml:"assessment_type"`
	TestingMode      string           `yaml:"testing_mode"`
	Classification   string           `yaml:"classification"`
	ReportVersion    string           `yaml:"report_version"`
	PreparedBy       string           `yaml:"prepared_by"`
	StartDate        string           `yaml:"start_date"`
	EndDate          string           `yaml:"end_date"`
	ExecutiveSummary string           `yaml:"executive_summary"`
	OverallPosture   string           `yaml:"overall_posture"`
	InScope          []string         `yaml:"in_scope"`
	OutOfScope       []string         `yaml:"out_of_scope"`
	Contacts         []string         `yaml:"contacts"`
	Restrictions     []string         `yaml:"restrictions"`
	EmergencyStop    []string         `yaml:"emergency_stop_conditions"`
	Limitations      []string         `yaml:"limitations"`
	Methodology      []string         `yaml:"methodology"`
	RevisionHistory  []reportRevision `yaml:"revision_history"`
}

type reportRevision struct {
	Version string `yaml:"version"`
	Date    string `yaml:"date"`
	Author  string `yaml:"author"`
	Changes string `yaml:"changes"`
}

func loadScope(root string) (engagementScope, error) {
	b, err := readFileIn(filepath.Join(root, markerFile))
	if err != nil {
		return engagementScope{}, err
	}
	var scope engagementScope
	if err := yaml.Unmarshal(b, &scope); err != nil {
		return engagementScope{}, fmt.Errorf("parse %s: %w", markerFile, err)
	}
	if scope.Engagement == "" {
		return engagementScope{}, fmt.Errorf("%s has no engagement name", markerFile)
	}
	return scope, nil
}
