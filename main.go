package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func main() {
	var (
		ignoreError bool
	)
	flag.BoolVar(&ignoreError, "i", false, "do not abort if an error occurs during run")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("missing argument: action")
	}

	action := flag.Arg(0)

	var err error
	switch action {
	case "clone":
		if flag.NArg() != 2 {
			log.Fatal("expected one argument: root project")
		}

		rootGroup := flag.Arg(1)

		err = clone(rootGroup)
	case "run":
		if flag.NArg() < 2 {
			log.Fatal("expected argument: command")
		}
		err = run(flag.Args()[1:], ignoreError)
	case "list":
		err = list()
	}

	if err != nil {
		log.Fatal(err)
	}
}

func clone(rootGroupName string) error {
	opts := []gitlab.ClientOptionFunc{}
	if server, found := os.LookupEnv("GITLAB_SERVER"); found {
		opts = append(opts, gitlab.WithBaseURL(server))
	}
	git, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), opts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	groups := []*gitlab.Group{}
	rootGroup, _, err := git.Groups.GetGroup(rootGroupName)
	if err != nil {
		return fmt.Errorf("could not get root group: %w", err)
	}
	groups = append(groups, rootGroup)
	descendantGroups, _, err := git.Groups.ListDescendantGroups(rootGroup.FullPath, &gitlab.ListDescendantGroupsOptions{})
	if err != nil {
		return fmt.Errorf("could not get descendant groups: %w", err)
	}

	groups = append(groups, descendantGroups...)

	for _, group := range groups {
		log.Printf("check %s", group.FullPath)
		projs, _, err := git.Groups.ListGroupProjects(group.FullPath, &gitlab.ListGroupProjectsOptions{})
		if err != nil {
			return fmt.Errorf("failed to get projects of group: %w", err)
		}

		for _, p := range projs {
			checkoutPath := p.PathWithNamespace[len(rootGroupName+"/"):]
			cmd := exec.Command("git", "clone", p.SSHURLToRepo, checkoutPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to run git clone %s %s: %w", p.SSHURLToRepo, checkoutPath, err)
			}
		}
	}
	return nil
}

func list() error {
	dirs, err := getGitDirs(".")
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		fmt.Println(dir)
	}
	return nil
}

func run(args []string, ignoreError bool) error {
	dirs, err := getGitDirs(".")
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		cmdArgs := []string{"-C", dir}
		cmdArgs = append(cmdArgs, args...)
		log.Printf("run git %s", strings.Join(cmdArgs, " "))
		cmd := exec.Command("git", cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			if ignoreError {
				log.Printf("failed to run git %s", strings.Join(cmdArgs, " "))
				continue
			}
			return fmt.Errorf("failed to run git %s: %w", strings.Join(cmdArgs, " "), err)
		}
	}
	return err
}

func getGitDirs(root string) ([]string, error) {
	gitDirs := []string{}
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		currentDir := filepath.Join(root, path)
		gitDir := filepath.Join(currentDir, ".git")
		file, err := os.Stat(gitDir)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if os.IsNotExist(err) {
			return nil
		}
		if !file.IsDir() {
			return nil
		}
		gitDirs = append(gitDirs, currentDir)
		return filepath.SkipDir
	})
	return gitDirs, err
}
