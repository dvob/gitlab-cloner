package main

import (
	"flag"
	"log"
	"os"
	"os/exec"

	"github.com/xanzy/go-gitlab"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("expected one argument: root project")
	}
	rootGroupName := flag.Arg(0)

	opts := []gitlab.ClientOptionFunc{}
	if server, found := os.LookupEnv("GITLAB_SERVER"); found {
		opts = append(opts, gitlab.WithBaseURL(server))
	}
	git, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	groups := []*gitlab.Group{}
	rootGroup, _, err := git.Groups.GetGroup(rootGroupName)
	if err != nil {
		log.Fatal("could not get root group:", err)
	}
	groups = append(groups, rootGroup)
	descendantGroups, _, err := git.Groups.ListDescendantGroups(rootGroup.FullPath, &gitlab.ListDescendantGroupsOptions{})
	if err != nil {
		log.Fatal("could not get descendant groups", err)
	}

	groups = append(groups, descendantGroups...)

	for _, group := range groups {
		log.Printf("check %s", group.FullPath)
		projs, _, err := git.Groups.ListGroupProjects(group.FullPath, &gitlab.ListGroupProjectsOptions{})
		if err != nil {
			log.Fatal("failed to get projects of group", err)
		}

		for _, p := range projs {
			checkoutPath := p.PathWithNamespace[len(rootGroupName+"/"):]
			cmd := exec.Command("git", "clone", p.SSHURLToRepo, checkoutPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
