package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/matthewmueller/golly/api"
	"github.com/matthewmueller/golly/golang"
	"github.com/pkg/errors"
)

var (
	cli  = kingpin.New("golly", "Go to Javascript compiler")
	root = cli.Flag("root", "package root").String()

	buildCmd      = cli.Command("build", "build the packages")
	buildPackages = buildCmd.Arg("packages", "packages to build").Required().Strings()

	serveCmd      = cli.Command("serve", "development server")
	servePackages = serveCmd.Arg("packages", "packages to bundle").Required().Strings()
	servePort     = serveCmd.Flag("port", "port to serve from").Default("8080").String()

	runCmd  = cli.Command("run", "run a package")
	runFile = runCmd.Arg("file", "file to run").Required().String()

	// TODO: just have this be argv[1]
	// pkg   = cli.Flag("pkg", "package path").Required().String()
	// graph = cli.Flag("graph", "call graph").Bool()
)

func main() {
	ctx := signalContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	log.SetHandler(text.New(os.Stderr))

	command, err := cli.Parse(os.Args[1:])
	if err != nil {
		cli.FatalUsage(err.Error())
	}

	if *root == "" {
		dir, e := os.Getwd()
		if e != nil {
			log.Fatal(e.Error())
		}
		*root = dir
	}

	var e error
	switch command {
	case "build":
		e = build(ctx)
	case "serve":
		e = serve(ctx)
	case "run":
		e = run(ctx)
	}
	if e != nil {
		log.Fatal(e.Error())
	}
}

func run(ctx context.Context) error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }

	// filePath := path.Join(cwd, *runFile)

	result, err := api.Run(ctx, &api.RunSettings{
		ChromePath: os.Getenv("GOLLY_CHROME_PATH"),
		FilePath:   *runFile,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(result)

	return nil
}

func build(ctx context.Context) error {
	packages, err := getMains(*buildPackages)
	if err != nil {
		return err
	}
	for i, pkg := range packages {
		packages[i] = path.Join(os.Getenv("GOPATH"), "src", pkg)
	}

	// start := time.Now()
	compiler := golang.New()
	files, _, e := compiler.Compile(packages...)
	if e != nil {
		return errors.Wrap(e, "error building packages")
	}

	for _, file := range files {
		fmt.Println("---")
		fmt.Println(file.Name)
		fmt.Println("---")
		fmt.Println(file.Source)
		fmt.Println("===")
	}
	return nil
}

func serve(ctx context.Context) error {
	packages, err := getMains(*servePackages)
	if err != nil {
		return err
	}

	for i, pkg := range packages {
		packages[i] = path.Join(os.Getenv("GOPATH"), "src", pkg)
	}

	port, e := strconv.Atoi(*servePort)
	if e != nil {
		return errors.Wrap(e, "invalid port")
	}

	return api.Serve(ctx, &api.ServeSettings{
		Packages: packages,
		Port:     port,
	})
}

func signalContext(ctx context.Context, sig ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, sig...)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()

	return ctx
}

// GoMainDirs returns the file paths to the packages that are "main"
// packages, from the list of packages given. The list of packages can
// include relative paths, the special "..." Go keyword, etc.
//
// Sourced from:
// https://github.com/mitchellh/gox/blob/c9740af9c6574448fd48eb30a71f964014c7a837/go.go#L123
func getMains(packages []string) ([]string, error) {
	goCmd, err := exec.LookPath("go")
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, len(packages)+3)
	args = append(args, "list", "-f", "{{.Name}}|{{.ImportPath}}")
	args = append(args, packages...)

	output, err := execGo(goCmd, nil, "", args...)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(output))
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			log.Warnf("Bad line reading packages: %s", line)
			continue
		}

		if parts[0] == "main" {
			results = append(results, parts[1])
		}
	}

	return results, nil
}

func execGo(GoCmd string, env []string, dir string, args ...string) (string, error) {
	var stderr, stdout bytes.Buffer
	cmd := exec.Command(GoCmd, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if env != nil {
		cmd.Env = env
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%s\nStderr: %s", err, stderr.String())
		return "", err
	}

	return stdout.String(), nil
}