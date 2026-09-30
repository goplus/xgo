/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package projectdriver

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

var baseVersionRE = regexp.MustCompile(`(?:^|[^0-9])v?([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:-[0-9A-Za-z.-]+)?)`)

const driverV1Minimum = "1.8.0"

// Development builds claim driver v1's baseline capability.
const developmentDriverCapability = driverV1Minimum

func checkDriverXGoVersion(declared, current string) error {
	minimum, _ := normalizeXGoVersion(driverV1Minimum)
	required := driverV1Minimum
	if declared != "" {
		normalized, ok := normalizeXGoVersion(declared)
		if !ok {
			return checkRequiredXGo(declared, current)
		}
		if semver.Compare(normalized, minimum) >= 0 {
			required = declared
		}
	}
	err := checkRequiredXGo(required, current)
	if err != nil && required == driverV1Minimum {
		return fmt.Errorf("driver v1 requires XGo %s, but xgo build is %s", driverV1Minimum, describeDriverVersion(current))
	}
	return err
}

func checkRequiredXGo(required, current string) error {
	if required == "" {
		return nil
	}
	requiredSemver, ok := normalizeXGoVersion(required)
	if !ok {
		return fmt.Errorf("invalid required XGo version %q (xgo build %s)", required, describeDriverVersion(current))
	}
	currentSemver, ok := comparableDriverVersion(current)
	if !ok || semver.Compare(currentSemver, requiredSemver) < 0 {
		return fmt.Errorf("declaring module requires XGo %s, but xgo build is %s", required, describeDriverVersion(current))
	}
	return nil
}

func comparableDriverVersion(version string) (string, bool) {
	if isDevelopmentVersion(version) {
		return normalizeXGoVersion(developmentDriverCapability)
	}
	return comparableCurrentVersion(version)
}

func describeDriverVersion(version string) string {
	if isDevelopmentVersion(version) {
		return fmt.Sprintf("%s (driver capability %s)", version, developmentDriverCapability)
	}
	return version
}

func isDevelopmentVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version == "(devel)" || strings.HasSuffix(version, " devel") || module.IsPseudoVersion(version)
}

func normalizeXGoVersion(version string) (string, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if strings.Count(strings.SplitN(version, "-", 2)[0], ".") == 1 {
		parts := strings.SplitN(version, "-", 2)
		version = parts[0] + ".0"
		if len(parts) == 2 {
			version += "-" + parts[1]
		}
	}
	version = "v" + version
	return version, semver.IsValid(version)
}

func comparableCurrentVersion(version string) (string, bool) {
	if normalized, ok := normalizeXGoVersion(version); ok {
		return normalized, true
	}
	// Display-form versions may include a comparable semantic base. Unversioned
	// development builds are handled by comparableDriverVersion.
	match := baseVersionRE.FindStringSubmatch(version)
	if len(match) != 2 {
		return "", false
	}
	return normalizeXGoVersion(match[1])
}
