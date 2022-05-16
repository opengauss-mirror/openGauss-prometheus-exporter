// Copyright © 2020 Bin Liu <bin.liu@enmotech.com>

package exporter

import (
	"errors"
	"fmt"
	"github.com/blang/semver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"regexp"
	"strconv"
	"strings"
)

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// parseConstLabels turn param string into prometheus.Labels
func parseConstLabels(s string) prometheus.Labels {
	labels := make(prometheus.Labels)
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil
	}

	parts := strings.Split(s, ",")
	for _, p := range parts {
		keyValue := strings.Split(strings.TrimSpace(p), "=")
		if len(keyValue) != 2 {
			log.Errorf(`malformed labels format %q, should be "key=value"`, p)
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		if key == "" || value == "" {
			continue
		}
		labels[key] = value
	}
	if len(labels) == 0 {
		return nil
	}

	return labels
}

// parseCSV will turn a comma separated string into a []string
func parseCSV(s string) (tags []string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil
	}

	parts := strings.Split(s, ",")
	for _, p := range parts {
		if tag := strings.TrimSpace(p); len(tag) > 0 {
			tags = append(tags, tag)
		}
	}

	if len(tags) == 0 {
		return nil
	}
	return
}

func parseVersionSem(versionString string) (semver.Version, error) {
	version := parseVersion(versionString)
	if version != "" {
		return semver.ParseTolerant(version)
	}
	return semver.Version{},
		errors.New(fmt.Sprintln("Could not find a openGauss version in string:", versionString))
}

var (
	gaussDBVerRep   = regexp.MustCompile(`(GaussDB|MogDB)\s+Kernel\s+V(\w+)`)
	openGaussVerRep = regexp.MustCompile(`(openGauss|MogDB)\s+(\d+\.\d+.\d+)`)
	vastbaseVerRep  = regexp.MustCompile(`(Vastbase\s+G100)\s+V(\d+\.\d+)`)
)

func parseVersion(versionString string) string {
	versionString = strings.TrimSpace(versionString)
	if gaussDBVerRep.MatchString(versionString) {
		return parseGaussDBVersion(gaussDBVerRep.FindStringSubmatch(versionString))
	}
	if openGaussVerRep.MatchString(versionString) {
		return parseOpenGaussVersion(openGaussVerRep.FindStringSubmatch(versionString))
	}
	if vastbaseVerRep.MatchString(versionString) {
		return parseVastbaseVersion(vastbaseVerRep.FindStringSubmatch(versionString))
	}
	return ""
}

func parseOpenGaussVersion(subMatches []string) string {
	if len(subMatches) < 3 || subMatches[2] == "" {
		return ""
	}
	return subMatches[2]
}

func parseVastbaseVersion(subMatches []string) string {
	if len(subMatches) < 3 || subMatches[2] == "" {
		return ""
	}
	return subMatches[2]
}

func parseGaussDBVersion(subMatches []string) string {
	if len(subMatches) < 3 || subMatches[2] == "" {
		return ""
	}
	r := regexp.MustCompile(`(\d+)R(\d+)C(\d+)`).FindStringSubmatch(subMatches[2])
	if len(r) < 3 {
		return ""
	}
	r1, _ := strconv.Atoi(r[1])
	r2, _ := strconv.Atoi(r[2])
	r3, _ := strconv.Atoi(r[3])
	return fmt.Sprintf("%v.%v.%v", r1, r2, r3)
}
