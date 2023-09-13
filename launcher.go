package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

//goland:noinspection GoSnakeCaseUsage
const (
	URL_VERSION_MANIFEST string = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
	URL_RESOURCES        string = "https://resources.download.minecraft.net/"
)

type VersionInfo struct {
	Id              string `json:"id"`
	Type            string `json:"type"`
	Url             string `json:"url"`
	Time            string `json:"time"`
	ReleaseTime     string `json:"releaseTime"`
	Sha1            string `json:"sha1"`
	ComplianceLevel int32  `json:"complianceLevel"`
}

func (this *VersionInfo) url() string {
	return this.Url
}

func (this *VersionInfo) hash() *string {
	return &this.Sha1
}

type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []VersionInfo `json:"versions"`
}

type Rule struct {
	Action   string          `json:"action"`
	Features map[string]bool `json:"features"`
	Os       struct {
		Arch string `json:"arch"`
		Name string `json:"name"`
	} `json:"os"`
}

func testRules(rules []Rule, features map[string]bool) bool {
	if len(rules) == 0 {
		return true
	}

	action := "disallow"

	for i := range rules {
		rule := rules[i]
		if rule.testRule(features) {
			action = rule.Action
		}
	}

	return action == "allow"
}

func (this *Rule) testRule(features map[string]bool) bool {
	for ruleFeature := range this.Features {
		value, ok := features[ruleFeature]
		if !ok {
			return false
		}
		if value != this.Features[ruleFeature] {
			return false
		}
	}

	if this.Os.Arch != "" && runtime.GOARCH != this.Os.Arch {
		return false
	}
	if this.Os.Name != "" && runtime.GOOS != this.Os.Name {
		return false
	}

	return true
}

type Artifact struct {
	Path string `json:"path"`
	Sha1 string `json:"sha1"`
	Size uint64 `json:"size"`
	Url  string `json:"url"`
}

func (this *Artifact) url() string {
	return this.Url
}

func (this *Artifact) hash() *string {
	return &this.Sha1
}

type Library struct {
	Downloads struct {
		Artifact Artifact `json:"artifact"`
	}
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Argument struct {
	Value []string `json:"value"`
	Rules []Rule   `json:"rules"`
}

func (this *Argument) UnmarshalJSON(bytes []byte) error {
	var raw interface{}
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return err
	}
	switch raw.(type) {
	case string:
		{
			this.Value = append(this.Value, raw.(string))
		}

	case map[string]interface{}:
		{
			object := raw.(map[string]interface{})
			rawRules, ok := object["rules"]
			if ok {
				rules := rawRules.([]interface{})
				ruleCount := len(rules)
				for i := 0; i < ruleCount; i++ {
					rawRule := rules[i].(map[string]interface{})
					var rule Rule

					rule.Action, ok = rawRule["action"].(string)
					if !ok {
						return errors.New("rule had no action")
					}

					rawFeatures, ok := rawRule["features"].(map[string]interface{})
					if ok {
						rule.Features = map[string]bool{}
						for key := range rawFeatures {
							rule.Features[key], ok = rawFeatures[key].(bool)
							if !ok {
								return errors.New("failed to convert rules features")
							}
						}
					}

					rawOs, ok := rawRule["os"].(map[string]interface{})
					if ok {
						arch, ok := rawOs["arch"].(string)
						if ok {
							rule.Os.Arch = arch
						}

						name, ok := rawOs["name"].(string)
						if ok {
							rule.Os.Name = name
						}
					}

					this.Rules = append(this.Rules, rule)
				}
			}

			rawValue, ok := object["value"]
			if ok {
				switch rawValue.(type) {
				case string:
					{
						this.Value = append(this.Value, rawValue.(string))
					}

				case []interface{}:
					{
						rawValues := rawValue.([]interface{})
						valueCount := len(rawValues)
						for i := 0; i < valueCount; i++ {
							this.Value = append(this.Value, rawValues[i].(string))
						}
					}
				}
			} else {
				return errors.New("rule had no value")
			}
		}

	default:
		{
			return errors.New(fmt.Sprintf("can't handle argument JSON: %s", string(bytes)))
		}
	}
	return nil
}

type AssetIndex struct {
	Id        string `json:"id"`
	Sha1      string `json:"sha1"`
	Size      uint64 `json:"size"`
	TotalSize uint64 `json:"totalSize"`
	Url       string `json:"url"`
}

func (this *AssetIndex) url() string {
	return this.Url
}

func (this *AssetIndex) hash() *string {
	return &this.Sha1
}

type Manifest struct {
	Arguments struct {
		Game []Argument `json:"game"`
		Jvm  []Argument `json:"jvm"`
	} `json:"arguments"`
	AssetIndex      AssetIndex `json:"assetIndex"`
	Assets          string     `json:"assets"`
	ComplianceLevel uint32     `json:"complianceLevel"`
	Downloads       map[string]struct {
		Sha1 string `json:"sha1"`
		Size uint64 `json:"size"`
		Url  string `json:"url"`
	} `json:"downloads"`
	Id          string `json:"id"`
	JavaVersion struct {
		Component    string `json:"component"`
		MajorVersion uint32 `json:"majorVersion"`
	} `json:"javaVersion"`
	Libraries []Library `json:"libraries"`
	Logging   map[string]struct {
		Argument string `json:"argument"`
		File     struct {
			Id   string `json:"id"`
			Sha1 string `json:"sha1"`
			Size uint64 `json:"size"`
			Url  string `json:"url"`
		} `json:"file"`
		Type string `json:"type"`
	} `json:"logging"`
	MainClass              string `json:"mainClass"`
	MinimumLauncherVersion uint32 `json:"minimumLauncherVersion"`
	ReleaseTime            string `json:"releaseTime"`
	Time                   string `json:"time"`
	Type                   string `json:"type"`
}

type AssetEntry struct {
	Hash string `json:"hash"`
	Size uint64 `json:"size"`
}

func (this *AssetEntry) url() string {
	return URL_RESOURCES + this.Hash[0:2] + "/" + this.Hash
}

func (this *AssetEntry) hash() *string {
	return &this.Hash
}

type AssetManifest struct {
	Objects map[string]AssetEntry `json:"objects"`
}

func downloadVersionManifest(manifest *VersionManifest) error {
	return downloadJsonRaw(URL_VERSION_MANIFEST, nil, manifest)
}

func downloadManifest(versions *VersionManifest, version string, manifest *Manifest) error {
	for i := range versions.Versions {
		current := versions.Versions[i]
		if current.Id == version {
			return downloadJson(&current, manifest)
		}
	}
	return errors.New("failed to find version manifest url for version " + version)
}

func jankyFormat(argument string, environment map[string]string) string {
	for {
		start := strings.Index(argument, "${")
		if start == -1 {
			return argument
		}

		end := strings.Index(argument, "}")
		value, ok := environment[argument[start+2:end]]
		if !ok { //TODO Graceful error handling
			return argument
		}

		argument = argument[:start] + value + argument[end+1:]
	}
}

func main() {
	base, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get cwd: %s\n", err)
		return
	}

	var versionManifest VersionManifest
	err = downloadVersionManifest(&versionManifest)
	if err != nil {
		fmt.Printf("Failed to download version manifest: %s\n", err)
		return
	}

	var manifest Manifest
	err = downloadManifest(&versionManifest, versionManifest.Latest.Release, &manifest)
	if err != nil {
		fmt.Printf("Failed to download manifest: %s\n", err)
		return
	}

	features := map[string]bool{}
	features["is_demo_user"] = false
	features["has_custom_resolution"] = true
	features["has_quick_plays_support"] = false
	features["is_quick_play_singleplayer"] = false
	features["is_quick_play_multiplayer"] = false
	features["is_quick_play_realms"] = false

	classpath, err := downloadLibraries(base, manifest.Libraries, features)
	if err != nil {
		fmt.Printf("Failed to download libraries: %s", err)
		return
	}

	err = downloadAssets(base, manifest)
	if err != nil {
		fmt.Printf("Failed to download assets: %s", err)
		return
	}

	jar := base + "/client/" + manifest.Id + ".jar"
	hash := manifest.Downloads["client"].Sha1
	err = downloadFileRaw(jar, manifest.Downloads["client"].Url, &hash)
	if err != nil {
		fmt.Printf("Failed to download client: %s", err)
		return
	}

	var command []string
	command = nil

	cp := jar
	for i := range classpath {
		cp = cp + ":" + classpath[i]
	}

	environment := map[string]string{}
	environment["natives_directory"] = "natives"
	environment["launcher_name"] = "PickAName"
	environment["launcher_version"] = "0.0.0"
	environment["classpath"] = cp
	environment["auth_player_name"] = "todo_name"
	environment["version_name"] = manifest.Id
	environment["game_directory"] = "run"
	environment["assets_root"] = base + "/assets"
	environment["assets_index_name"] = manifest.AssetIndex.Id
	environment["auth_uuid"] = "00000000-0000-0000-0000-000000000000"
	environment["clientid"] = "0"
	environment["auth_xuid"] = "0"
	environment["auth_access_token"] = "0"
	environment["user_type"] = "asdf"
	environment["version_type"] = manifest.Type
	environment["resolution_width"] = "800"
	environment["resolution_height"] = "800"
	environment["quickPlayPath"] = "asdf"
	environment["quickPlaySingleplayer"] = "asdf"
	environment["quickPlayMultiplayer"] = "asdf"
	environment["quickPlayRealms"] = "asdf"

	for index := range manifest.Arguments.Jvm {
		argument := manifest.Arguments.Jvm[index]
		if testRules(argument.Rules, features) {
			for o := range argument.Value {
				command = append(command, jankyFormat(argument.Value[o], environment))
			}
		}
	}

	command = append(command, manifest.MainClass)

	for index := range manifest.Arguments.Game {
		argument := manifest.Arguments.Game[index]
		if testRules(argument.Rules, features) {
			for o := range argument.Value {
				command = append(command, jankyFormat(argument.Value[o], environment))
			}
		}
	}

	process := exec.Command("/lib/jvm/jdk-17.0.5+8/bin/java", command...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	process.Start()
	process.Wait()
}

func downloadAssets(base string, version Manifest) error {
	jsonPath := base + "/assets/indexes/" + version.AssetIndex.Id + ".json"
	err := downloadFile(jsonPath, &version.AssetIndex)
	if err != nil {
		return errors.Join(errors.New("failed to download asset manifest"), err)
	}

	file, err := os.Open(jsonPath)
	if err != nil {
		return errors.Join(errors.New("failed to open assets file"), err)
	}

	buffer, err := io.ReadAll(file)
	if err != nil {
		return errors.Join(errors.New("failed to read assets file"), err)
	}

	var manifest AssetManifest
	err = json.Unmarshal(buffer, &manifest)
	if err != nil {
		return errors.Join(errors.New("failed to parse assets file"), err)
	}

	channel := make(chan error)
	downloaded := map[string]bool{}
	for key := range manifest.Objects {
		object := manifest.Objects[key]
		if downloaded[object.Hash] {
			continue
		}

		downloaded[object.Hash] = true
		go func(base string, entry AssetEntry, channel chan error) {
			file := entry.Hash[0:2] + "/" + entry.Hash
			path := base + "/assets/objects/" + file
			channel <- downloadFile(path, &entry)
		}(base, object, channel)
	}

	err = nil
	length := len(downloaded)
	for i := 0; i < length; i++ {
		err = errors.Join(err, <-channel)
	}

	return err
}

func downloadLibraries(base string, libraries []Library, features map[string]bool) ([]string, error) {
	length := len(libraries)
	if length == 0 {
		return nil, nil
	}

	var classpath []string
	channel := make(chan error)
	for i := 0; i < length; i++ {
		library := libraries[i]

		if !testRules(library.Rules, features) {
			continue
		}

		path := base + "/library/" + library.Downloads.Artifact.Path
		classpath = append(classpath, path)

		go func(path string, lib Library) {
			channel <- downloadFile(path, &library.Downloads.Artifact)
		}(path, library)
	}

	var err error
	err = nil
	length = len(classpath)
	for i := 0; i < length; i++ {
		err = errors.Join(err, <-channel)
	}
	if err != nil {
		return nil, err
	}
	return classpath, nil
}
