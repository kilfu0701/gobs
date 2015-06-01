package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

var cmdInit = &Command{
	UsageLine: "init [SafariExt] [-file=NAME]",
	Short:     "init a build script.",
	Long: `Initialize a build script.
Usage:

  * Initial a script named "mySafariExt"

    gobs init SafariExt -file=my_safari_ext

`,
	CustomFlags: false,
}

var fileName string

func init() {
	cmdInit.Run = runInit
	cmdInit.Flag.StringVar(&fileName, "file", "MySafariExt", "output filename.")
}

func runInit(cmd *Command, args []string) int {
	allCmd := map[string]string{
		"SafariExt": `{
    "id": "com.myfirst.safariextension",
    "name": "%s",
    "developer_id": "1234ABCD",
    "path": {
        "tmp": ".tmp",
        "src": "path/to/%s.safariextension",
        "dist": "output/Safari",
        "certs": "path/to/certs",
        "l10n": "L10n"
    },
    "locales": {
        "en-US": "https://domain.com/en-US/Update.plist",
        "ja-JP": "https://domain.com/ja-JP/Update.plist",
        "zh-TW": "https://domain.com/zh-TW/Update.plist"
    },
    "bin": {
        "xar":     "/usr/local/bin/xar",
        "openssl": "/usr/bin/openssl"
    },
    "author": "",
    "description": "",
    "lincese": "",
    "gobs_version": "` + version + `",
    "gobs_type": "SafariExt"
}`,
	}

	allCmdHelp := map[string]string{
		"SafariExt": `Please set needed files into these directories...

@extract certs
    mkdir certs
    /usr/local/bin/xar -f my.safariextz --extract-certs certs
    openssl pkcs12 -in Certificates.p12 -nodes | openssl x509 -outform der -out cert.der
    openssl pkcs12 -in Certificates.p12 -nodes | openssl rsa -out key.pem
    openssl dgst -sign key.pem -binary < key.pem | wc -c > size.txt

    ( More details @ https://gist.github.com/kilfu0701/8e0ee7e856d1960c7ff9 ,
                   @ http://developer.streak.com/2013/01/how-to-build-safari-extension-using.html )

@certs
    path/to/certs/cert.dem
    path/to/certs/cert.00
    path/to/certs/cert.01
    path/to/certs/cert.02
    path/to/certs/key.pem
    path/to/certs/size.txt

@bin
    set executable binary path for "xar" & "openssl"

@locales
    L10n/en-US/messages.json
    L10n/ja-JP/messages.json

    json basic format =>
    {
        "extName" : {
            "message" : "My First Safari Extension"
        },
        "extDescription" : {
            "message" : "Try it, you'll love it."
        },
        "extTitle" : {
            "message" : "An Awesom Extension"
        }
    }`,
	}

	if len(args) != 0 {
		scriptType := args[0]
		cmd.Flag.Parse(args[1:])

		if len(allCmd[scriptType]) > 0 {
			fmt.Println("Start init: \t", scriptType)

			path, _ := os.Getwd()
			outputPath := path + "/" + fileName + ".json"

			jsonString := fmt.Sprintf(allCmd[scriptType], fileName, fileName)
			buf := []byte(jsonString)
			err := ioutil.WriteFile(outputPath, buf, 0644)

			if err != nil {
				panic(err)
			} else {
				fmt.Println("Init finished:\t", outputPath)
				fmt.Println("-----")
				fmt.Println(allCmdHelp[scriptType])
			}

		}

	} else {
		panic("Parameter wrong.")
	}

	return 0
}
