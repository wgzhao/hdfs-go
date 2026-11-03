package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/colinmarc/hdfs/v2"
	"github.com/colinmarc/hdfs/v2/hadoopconf"
	"github.com/pborman/getopt"
)

// TODO: tree, expunge/trash

var (
	version string
	usage   = fmt.Sprintf(`Usage: %s COMMAND
The flags available are a subset of the POSIX ones, but should behave similarly.

Valid commands:
  ls [-lahR] [FILE]...
  rm [-rf] FILE...
  mv [-nT] SOURCE... DEST
  mkdir [-p] FILE...
  touch [-c] FILE...
  touchz FILE...
  chmod [-R] OCTAL-MODE FILE...
  chown [-R] OWNER[:GROUP] FILE...
  chgrp [-R] GROUP FILE...
  cat SOURCE...
  head [-n LINES | -c BYTES] SOURCE...
  tail [-n LINES | -c BYTES] SOURCE...
  test [-defsz] FILE...
  du [-sh] FILE...
  df [-h]
  count [-qhvue] FILE...
  stat [FORMAT] FILE...
  find [FILE...] [-name PATTERN] [-iname PATTERN] [-print | -print0]
  checksum FILE...
  text [-ignoreCrc] SOURCE...
  get SOURCE... DEST
  getmerge [-n] SOURCE DEST
  copyToLocal SOURCE... DEST
  put SOURCE... DEST
  copyFromLocal SOURCE... DEST
  moveFromLocal SOURCE... DEST
  moveToLocal SOURCE DEST
  appendToFile SOURCE... DEST
  cp [-fpd] SOURCE... DEST
  rmdir [--ignore-fail-on-non-empty] DIR...
  truncate SIZE FILE
  setrep [-Rw] REPLICATION FILE...
  createSnapshot DIR [NAME]
  deleteSnapshot DIR NAME
  renameSnapshot DIR OLD NEW
  getfacl [-R] PATH...
  setfacl [-R] {-b|-k} [-m|-x ACL] PATH...
  setfacl [-R] --set ACL PATH...
  getfattr [-R] {-n NAME | -d} [-e ENC] PATH...
  setfattr {-n NAME [-v VALUE] | -x NAME} PATH...
  usage [COMMAND...]
`, os.Args[0])

	lsOpts = getopt.New()
	lsl    = lsOpts.Bool('l')
	lsa    = lsOpts.Bool('a')
	lsh    = lsOpts.Bool('h')
	lsR    = lsOpts.Bool('R')

	rmOpts = getopt.New()
	rmr    = rmOpts.Bool('r')
	rmf    = rmOpts.Bool('f')

	mvOpts = getopt.New()
	mvn    = mvOpts.Bool('n')
	mvT    = mvOpts.Bool('T')

	mkdirOpts = getopt.New()
	mkdirp    = mkdirOpts.Bool('p')

	testOpts = getopt.New()
	teste    = testOpts.Bool('e')
	testf    = testOpts.Bool('f')
	testd    = testOpts.Bool('d')
	testz    = testOpts.Bool('z')
	tests    = testOpts.Bool('s')

	touchOpts = getopt.New()
	touchc    = touchOpts.Bool('c')

	chmodOpts = getopt.New()
	chmodR    = chmodOpts.Bool('R')

	chownOpts = getopt.New()
	chownR    = chownOpts.Bool('R')

	headTailOpts = getopt.New()
	headtailn    = headTailOpts.Int64('n', -1)
	headtailc    = headTailOpts.Int64('c', -1)

	duOpts = getopt.New()
	dus    = duOpts.Bool('s')
	duh    = duOpts.Bool('h')

	getmergeOpts = getopt.New()
	getmergen    = getmergeOpts.Bool('n')

	dfOpts = getopt.New()
	dfh    = dfOpts.Bool('h')

	chgrpOpts = getopt.New()
	chgrpR    = chgrpOpts.Bool('R')

	countOpts = getopt.New()
	countQ    = countOpts.Bool('q')
	countH    = countOpts.Bool('h')
	countV    = countOpts.Bool('v')
	countU    = countOpts.Bool('u')
	countE    = countOpts.Bool('e')

	cpOpts = getopt.New()
	cpf    = cpOpts.Bool('f')
	cpp    = cpOpts.Bool('p')
	cpd    = cpOpts.Bool('d')

	getfaclOpts = getopt.New()
	getfaclR    = getfaclOpts.Bool('R')

	getfattrOpts = getopt.New()
	getfattrR    = getfattrOpts.Bool('R')
	getfattrN    = getfattrOpts.String('n', "")
	getfattrD    = getfattrOpts.Bool('d')
	getfattrE    = getfattrOpts.String('e', "text")

	setfattrOpts = getopt.New()
	setfattrN    = setfattrOpts.String('n', "")
	setfattrV    = setfattrOpts.String('v', "")
	setfattrX    = setfattrOpts.String('x', "")

	rmdirOpts   = getopt.New()
	rmdirIgnore = rmdirOpts.BoolLong("ignore-fail-on-non-empty", 0)

	setrepOpts = getopt.New()
	setrepR    = setrepOpts.Bool('R')
	setrepW    = setrepOpts.Bool('w')

	cachedClients map[string]*hdfs.Client = make(map[string]*hdfs.Client)
	status                                = 0
)

func init() {
	lsOpts.SetUsage(printHelp)
	rmOpts.SetUsage(printHelp)
	mvOpts.SetUsage(printHelp)
	touchOpts.SetUsage(printHelp)
	chmodOpts.SetUsage(printHelp)
	chownOpts.SetUsage(printHelp)
	headTailOpts.SetUsage(printHelp)
	duOpts.SetUsage(printHelp)
	getmergeOpts.SetUsage(printHelp)
	dfOpts.SetUsage(printHelp)
	testOpts.SetUsage(printHelp)
	chgrpOpts.SetUsage(printHelp)
	countOpts.SetUsage(printHelp)
	cpOpts.SetUsage(printHelp)
	getfaclOpts.SetUsage(printHelp)
	getfattrOpts.SetUsage(printHelp)
	setfattrOpts.SetUsage(printHelp)
	rmdirOpts.SetUsage(printHelp)
	setrepOpts.SetUsage(printHelp)
}

func main() {
	if len(os.Args) < 2 {
		fatalWithUsage()
	}

	command := os.Args[1]
	argv := os.Args[1:]
	switch command {
	case "-v", "--version":
		fmt.Println("gohdfs version", version)
		os.Exit(0)
	case "ls":
		lsOpts.Parse(argv)
		ls(lsOpts.Args(), *lsl, *lsa, *lsh, *lsR)
	case "rm":
		rmOpts.Parse(argv)
		rm(rmOpts.Args(), *rmr, *rmf)
	case "mv":
		mvOpts.Parse(argv)
		mv(mvOpts.Args(), !*mvn, *mvT)
	case "mkdir":
		mkdirOpts.Parse(argv)
		mkdir(mkdirOpts.Args(), *mkdirp)
	case "touch":
		touchOpts.Parse(argv)
		touch(touchOpts.Args(), *touchc)
	case "chown":
		chownOpts.Parse(argv)
		chown(chownOpts.Args(), *chownR)
	case "chmod":
		chmodOpts.Parse(argv)
		chmod(chmodOpts.Args(), *chmodR)
	case "chgrp":
		chgrpOpts.Parse(argv)
		chgrp(chgrpOpts.Args(), *chgrpR)
	case "cat":
		cat(argv[1:])
	case "appendToFile":
		appendToFile(argv[1:])
	case "copyFromLocal":
		put(argv[1:])
	case "copyToLocal":
		get(argv[1:])
	case "head", "tail":
		headTailOpts.Parse(argv)
		printSection(headTailOpts.Args(), *headtailn, *headtailc, (command == "tail"))
	case "du":
		duOpts.Parse(argv)
		du(duOpts.Args(), *dus, *duh)
	case "checksum":
		checksum(argv[1:])
	case "get":
		get(argv[1:])
	case "getmerge":
		getmergeOpts.Parse(argv)
		getmerge(getmergeOpts.Args(), *getmergen)
	case "put":
		put(argv[1:])
	case "moveFromLocal":
		moveFromLocal(argv[1:])
	case "moveToLocal":
		moveToLocal(argv[1:])
	case "cp":
		cpOpts.Parse(argv)
		// -d is accepted for compatibility; no ._COPYING_ file is created.
		_ = cpd
		cp(cpOpts.Args(), *cpf, *cpp)
	case "rmdir":
		rmdirOpts.Parse(argv)
		rmdir(rmdirOpts.Args(), *rmdirIgnore)
	case "count":
		countOpts.Parse(argv)
		count(countOpts.Args(), *countQ, *countH, *countV, *countU, *countE)
	case "stat":
		stat(argv[1:])
	case "find":
		find(argv[1:])
	case "text":
		text(stripIgnoreCrcFlag(argv[1:]))
	case "touchz":
		touchz(argv[1:])
	case "setrep":
		setrepOpts.Parse(argv)
		setrep(setrepOpts.Args(), *setrepR, *setrepW)
	case "createSnapshot":
		createSnapshot(argv[1:])
	case "deleteSnapshot":
		deleteSnapshot(argv[1:])
	case "renameSnapshot":
		renameSnapshot(argv[1:])
	case "getfacl":
		getfaclOpts.Parse(argv)
		getfacl(getfaclOpts.Args(), *getfaclR)
	case "setfacl":
		setfacl(argv[1:])
	case "getfattr":
		getfattrOpts.Parse(argv)
		getfattr(getfattrOpts.Args(), *getfattrR, *getfattrN, *getfattrD, *getfattrE)
	case "setfattr":
		setfattrOpts.Parse(argv)
		setfattr(setfattrOpts.Args(), *setfattrN, *setfattrV, *setfattrX)
	case "df":
		dfOpts.Parse(argv)
		df(*dfh)
	case "test":
		testOpts.Parse(argv)
		test(testOpts.Args(), *teste, *testf, *testd, *testz, *tests)
	case "truncate":
		truncate(argv[1:])
	case "usage":
		printUsage(argv[1:])
	// it's a seeeeecret command
	case "complete":
		complete(argv)
	case "help", "-h", "-help", "--help":
		printHelp()
	default:
		fatalWithUsage("Unknown command:", command)
	}

	os.Exit(status)
}

func printHelp() {
	fmt.Fprintln(os.Stderr, usage)
	os.Exit(0)
}

func printUsage(args []string) {
	if len(args) == 0 {
		printHelp()
	}

	lines := strings.Split(usage, "\n")
	for _, arg := range args {
		name := strings.TrimPrefix(arg, "-")
		found := false
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == name {
				fmt.Println(strings.TrimSpace(line))
				found = true
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", arg)
			status = 1
		}
	}
}

func stripIgnoreCrcFlag(args []string) []string {
	for len(args) > 0 && (args[0] == "-ignoreCrc" || args[0] == "--ignoreCrc") {
		args = args[1:]
	}

	return args
}

func fatal(msg ...interface{}) {
	fmt.Fprintln(os.Stderr, msg...)
	os.Exit(1)
}

func fatalWithUsage(msg ...interface{}) {
	if len(msg) > 0 {
		fmt.Fprintln(os.Stderr, append(msg, "\n"+usage)...)
	} else {
		fmt.Fprintln(os.Stderr, usage)
	}

	os.Exit(2)
}

func getClient(namenode string) (*hdfs.Client, error) {
	if cachedClients[namenode] != nil {
		return cachedClients[namenode], nil
	}

	if namenode == "" {
		namenode = os.Getenv("HADOOP_NAMENODE")
	}

	conf, err := hadoopconf.LoadFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("Problem loading configuration: %s", err)
	}

	options := hdfs.ClientOptionsFromConf(conf)
	if namenode != "" {
		options.Addresses = strings.Split(namenode, ",")
	}

	if options.Addresses == nil {
		return nil, errors.New("Couldn't find a namenode to connect to. You should specify hdfs://<namenode>:<port> in your paths. Alternatively, set HADOOP_NAMENODE or HADOOP_CONF_DIR in your environment.")
	}

	if options.KerberosClient != nil {
		options.KerberosClient, err = getKerberosClient()
		if err != nil {
			return nil, fmt.Errorf("Problem with kerberos authentication: %s", err)
		}
	} else {
		options.User = os.Getenv("HADOOP_USER_NAME")
		if options.User == "" {
			u, err := user.Current()
			if err != nil {
				return nil, fmt.Errorf("Couldn't determine user: %s", err)
			}

			options.User = u.Username
		}
	}

	// Set some basic defaults.
	dialFunc := (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 5 * time.Second,
		DualStack: true,
	}).DialContext

	options.NamenodeDialFunc = dialFunc
	options.DatanodeDialFunc = dialFunc

	c, err := hdfs.NewClient(options)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to namenode: %s", err)
	}

	cachedClients[namenode] = c
	return c, nil
}
