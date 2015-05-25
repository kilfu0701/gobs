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

	path, _ := os.Getwd()
	extName := dat["name"].(string)

	fmt.Println("  Name  \t=", extName)

	// parse 'path'
	pathObject := dat["path"].(map[string]interface{})
	pathTmp := path + "/" + pathObject["tmp"].(string)
	pathSrc := path + "/" + pathObject["src"].(string)
	pathDist := path + "/" + pathObject["dist"].(string)
	pathCerts := path + "/" + pathObject["certs"].(string)
	pathL10n := path + "/" + pathObject["l10n"].(string)

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
	for locale := range locales {
		fmt.Println("  locale \t=>", locale, pathL10n+"/"+locale+"/messages.json")

		f, err := ioutil.ReadFile(pathL10n + "/" + locale + "/messages.json")
		if err != nil {
			panic(err)
		}

		byt := []byte(string(f))
		var dat map[string]interface{}
		if err := json.Unmarshal(byt, &dat); err != nil {
			panic(err)
		}

		packageName := dat["extName"].(map[string]interface{})["message"].(string)
		packageDescription := dat["extDescription"].(map[string]interface{})["message"].(string)

		lines := readLine(pathSrc + "/Info.plist")

		for i := 0; i < len(lines); i++ {
			s := strings.TrimSpace(lines[i])
			if s == "<key>CFBundleDisplayName</key>" {
				lines[i+1] = "\t<string>" + packageName + "</string>"
			} else if s == "<key>CFBundleShortVersionString</key>" {
				lines[i+1] = "\t<string>" + buildVersion + "</string>"
			} else if s == "<key>Description</key>" {
				lines[i+1] = "\t<string>" + packageDescription + "</string>"
			}
		}

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
		buildDir := pathDist + "/" + buildVersion + "/" + locale
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

		fmt.Println("** Build Success  =>", dest)

		os.Chdir(path)
	}

	return 0
}
