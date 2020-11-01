package main

import (
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strings"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/ssh"
)

//Args - arguments to cscp
type Args struct {
	username   string
	hostname   string
	preserve   bool
	recursive  bool
	port       string
	localPath  string
	remotePath string
	push       bool
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func argparser() *Args {
	sargs := &Args{port: "22", push: false, recursive: false, preserve: false}

	uRegex := regexp.MustCompile(".*@.*")

	remotePathRegex := regexp.MustCompile(".*@.*:.*")
	switchRegex := regexp.MustCompile("-.*")
	args := os.Args[1:]
	skip := make(map[int]struct{})
	for i, arg := range args {
		if _, ok := skip[i]; !ok {
			if uRegex.MatchString(arg) {
				//username & hostname
				sargs.username = strings.Split(arg, "@")[0]
				sargs.hostname = strings.Split(strings.Split(arg, "@")[1], ":")[0]
			}
			if remotePathRegex.MatchString(arg) {
				//remote path of file or directory
				sargs.remotePath = strings.Split(arg, ":")[1]
				sargs.push = true
			} else if switchRegex.MatchString(arg) {
				//this is one switch or a combination of switches
				if strings.Contains(arg, "p") {
					sargs.preserve = true
				} else if strings.Contains(arg, "r") {
					sargs.recursive = true
				} else if strings.Contains(arg, "P") {
					//get the next arg and assume it's the port
					sargs.port = args[i+1]
					//add next index to skip hash
					skip[i+1] = struct{}{}
				}
			} else {
				if fileExists(arg) {
					//must be a local path
					sargs.localPath = arg
				} else {
					//local file doesn't exist
					panic(fmt.Errorf("ERROR: %s does not exist on the local filesystem", arg))
				}
			}
		}
	}
	return sargs
}

func main() {

	args := argparser()
	spew.Dump(args)

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	clientConfig, _ := auth.PrivateKey(args.username, usr.HomeDir+"/.ssh/id_rsa", ssh.InsecureIgnoreHostKey())

	// Create a new SCP client
	client := scp.NewClient(args.hostname+":"+args.port, &clientConfig)

	// Connect to the remote server
	err = client.Connect()
	if err != nil {
		fmt.Println("Couldn't establish a connection to the remote server ", err)
		return
	}

	// Open a file
	f, _ := os.Open(args.localPath)
	// Compress it as it comes in
	// _ = lz4.NewWriter(f)
	// Close client connection after the file has been copied
	defer client.Close()

	// Close the file after it has been copied
	defer f.Close()

	// Copy
	err = client.CopyFile(f, args.remotePath, "0655")

	if err != nil {
		fmt.Println("Error while copying file ", err)
	}
}
