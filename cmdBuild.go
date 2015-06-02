package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var cmdBuild = &Command{
	UsageLine: "build [$SCRIPT_NAME] [-version=123]",
	Short:     "build a simply process.",
	Long: `Simple build process.
Usage:

  * Build a Safari extension:

    gobs build SafariExt.json -version=123

`,
}

var buildVersion string

func init() {
	cmdBuild.Run = runBuild
	cmdBuild.Flag.StringVar(&buildVersion, "version", "1", "version for build")
}

func runBuild(cmd *Command, args []string) int {
	//fmt.Println(args, buildVersion)

	if len(args) != 0 {
		scriptName := args[0]
		cmd.Flag.Parse(args[1:])

		path, _ := os.Getwd()

		// read build script file.
		configFile := path + "/" + scriptName
		f, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic("Config file not found ... => " + configFile)
		}

		byt := []byte(string(f))
		var dat map[string]interface{}
		if err := json.Unmarshal(byt, &dat); err != nil {
			panic(err)
		}

		scriptType := dat["gobs_type"]
		//dat["extDescription"].(map[string]interface{})["message"].(string)
		switch scriptType {
		case "SafariExt":
			return safariExt(dat)
			break

		default:
			fmt.Println("Unknow gobs_type =>", scriptType)
			break
		}

	} else {
		panic("Parameter Empty.")
	}

	return 0
}

func safariExt(dat map[string]interface{}) int {
	fmt.Println("Start building 'SafariExt' ...")

	UpdatePlistStr := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
   <key>Extension Updates</key>
   <array>
     <dict>
       <key>CFBundleIdentifier</key>
       <string>%s</string>
       <key>Developer Identifier</key>
       <string>%s</string>
       <key>CFBundleVersion</key>
       <string>%s</string>
       <key>CFBundleShortVersionString</key>
       <string>%s</string>
       <key>URL</key>
       <string>%s</string>
     </dict>
   </array>
</dict>
</plist>`

	path, _ := os.Getwd()
	id := dat["id"].(string)
	extName := dat["name"].(string)

	fmt.Println("  Name  \t=", extName)

	// parse 'path'
	pathObject := dat["path"].(map[string]interface{})

	pathTmp := pathObject["tmp"].(string)
	if string(pathTmp[0]) != "/" {
		pathTmp = path + "/" + pathTmp
	}

	pathSrc := pathObject["src"].(string)
	if string(pathSrc[0]) != "/" {
		pathSrc = path + "/" + pathSrc
	}

	pathDist := pathObject["dist"].(string)
	if string(pathDist[0]) != "/" {
		pathDist = path + "/" + pathDist
	}

	pathCerts := pathObject["certs"].(string)
	if string(pathCerts[0]) != "/" {
		pathCerts = path + "/" + pathCerts
	}

	pathL10n := path + "/" + pathObject["l10n"].(string)
	if string(pathL10n[0]) != "/" {
		pathL10n = path + "/" + pathL10n
	}

	fmt.Println("  pathTmp \t=", pathSrc)
	fmt.Println("  pathSrc \t=", pathSrc)
	fmt.Println("  pathDist \t=", pathDist)
	fmt.Println("  pathCerts \t=", pathCerts)
	fmt.Println("  pathL10n \t=", pathL10n)

	// parse 'bin'
	binObject := dat["bin"].(map[string]interface{})
	binXar := binObject["xar"].(string)
	binOpenssl := binObject["openssl"].(string)

	fmt.Println("  binXar \t=", binXar)
	fmt.Println("  binOpenssl \t=", binOpenssl)

	arr := strings.Split(pathSrc, "/")
	extFolderName := arr[len(arr)-1]

	// cleanup
	fmt.Println("* Cleanup ...", pathTmp)
	err := os.RemoveAll(pathTmp)

	// copy src to .tmp
	fmt.Println("* Copy files ...", pathSrc, "To", pathTmp+"/"+extFolderName)
	err = CopyDir(pathSrc, pathTmp+"/"+extFolderName)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print("* Files copied.")
	}

	locales := dat["locales"].(map[string]interface{})
	for locale, value := range locales {
		fmt.Println("  locale \t=>", locale, pathL10n+"/"+locale+"/messages.json")

		valueObj := value.(map[string]interface{})
		updateURL := valueObj["update_plist"].(string)
		updatePath := valueObj["update_path"].(string)

		f, err := ioutil.ReadFile(pathL10n + "/" + locale + "/messages.json")
		if err != nil {
			fmt.Println("Parse json error.")
			panic(err)
		}

		byt := []byte(string(f))
		var l10n_dat map[string]interface{}
		if err := json.Unmarshal(byt, &l10n_dat); err != nil {
			panic(err)
		}

		packageName := l10n_dat["extName"].(map[string]interface{})["message"].(string)
		packageDescription := l10n_dat["extDescription"].(map[string]interface{})["message"].(string)

		// start "Update Manifest"
		lines := readLine(pathSrc + "/Info.plist")
		hasUpdateURL := false
		for i := 0; i < len(lines); i++ {
			s := strings.TrimSpace(lines[i])
			if s == "<key>CFBundleDisplayName</key>" {
				lines[i+1] = "\t<string>" + packageName + "</string>"
			} else if s == "<key>CFBundleShortVersionString</key>" {
				lines[i+1] = "\t<string>" + buildVersion + "</string>"
			} else if s == "<key>Description</key>" {
				lines[i+1] = "\t<string>" + packageDescription + "</string>"
			} else if s == "<key>Tool Tip</key>" {
				lines[i+1] = "\t\t<string>" + packageName + "</string>"
			} else if s == "<key>Label</key>" {
				lines[i+1] = "\t\t<string>" + packageName + "</string>"
			} else if s == "<key>Update Manifest URL</key>" {
				lines[i+1] = "\t<string>" + updateURL + "</string>"
				hasUpdateURL = true
			}
		}

		if !hasUpdateURL {
			arr := make([]string, 2)
			arr[0] = "\t<key>Update Manifest URL</key>"
			arr[1] = "\t<string>" + updateURL + "</string>"

			lg := len(lines)
			lines = append(append(lines[:lg-4], arr...), lines[lg-2:]...)
		}
		// end of "Update Manifest"

		s := ""
		for _, v := range lines {
			s = s + v + "\r\n"
		}
		b := []byte(s)
		plist_path := pathTmp + "/" + extFolderName + "/Info.plist"
		os.Remove(plist_path)
		err = ioutil.WriteFile(plist_path, b, 0644)
		if err != nil {
			panic(err)
		}

		// pack
		// check if $VERSION in dist ?
		idx := strings.Index(pathDist, "$VERSION")
		buildDir := ""
		if idx == -1 {
			buildDir = pathDist + "/" + buildVersion + "/" + locale
		} else {
			buildDir = strings.Replace(pathDist, "$VERSION", buildVersion, -1) + "/" + locale
		}

		os.MkdirAll(buildDir, 0777)
		dest := buildDir + "/" + extName + ".safariextz"
		source := extFolderName

		// change dir to .tmp
		os.Chdir(pathTmp)
		//fmt.Println("chdir =>", pathTmp)

		_, err = exec.Command(
			binXar,
			"-czf",
			dest,
			"--distribution",
			source,
		).Output()

		if err != nil {
			panic(err)
		}

		// read size.txt
		f, err = ioutil.ReadFile(pathCerts + "/size.txt")
		size := strings.TrimSpace(string(f))

		_, err = exec.Command(
			binXar,
			"--sign",
			"-f",
			dest,
			"--digestinfo-to-sign",
			"digest.dat",
			"--sig-size",
			size,
			"--cert-loc",
			pathCerts+"/cert.der",
			"--cert-loc",
			pathCerts+"/cert01",
			"--cert-loc",
			pathCerts+"/cert02",
		).Output()

		if err != nil {
			panic(err)
		}

		_, err = exec.Command(
			binOpenssl,
			"rsautl",
			"-sign",
			"-inkey",
			pathCerts+"/key.pem",
			"-in",
			"digest.dat",
			"-out",
			"sig.dat",
		).Output()

		if err != nil {
			panic(err)
		}

		_, err = exec.Command(
			binXar,
			"--inject-sig",
			"sig.dat",
			"-f",
			dest,
		).Output()

		if err != nil {
			panic(err)
		}

		// write Update.plist
		if dat["developer_id"] == nil {
			panic("* ERROR * ... 'developer_id' not found in your BuildScript.json.")
		}

		versionNameWithoutDot := strings.Replace(buildVersion, ".", "", -1)
		buf := []byte(fmt.Sprintf(UpdatePlistStr, id, dat["developer_id"].(string), versionNameWithoutDot, buildVersion, updatePath))
		err = ioutil.WriteFile(buildDir+"/Update.plist", buf, 0644)
		if err != nil {
			panic(err)
		}

		fmt.Println("** Build Success  =>", dest)

		os.Chdir(path)
	}

	return 0
}
