# Contribution Guidelines

Excited by our work and want to get involved in building out our sharding releases? Or maybe you haven't learned as much about the Ethereum protocol but are a savvy developer? Our [READINGS.md](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/READINGS.md) doc includes comprehensive information on Ethereum and sharding for both part-time and core contributors to the project.

Additionally, our [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/README.md) serves source of truth for all things related to our implementation of sharding fo Ethereum.

You can explore our [Current Projects](https://github.com/prysmaticlabs/geth-sharding/projects) in-the works for our different releases. Feel free to fork our repo and start creating PR’s after assigning yourself to an issue of interest. We are always chatting on [Gitter](https://gitter.im/prysmaticlabs/geth-sharding), so drop us a line there if you want to get more involved or have any questions on our implementation!

**Contribution Steps**

-   Create a folder in your `$GOPATH` and navigate to it `mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum`
-   Clone our repository as `go-ethereum`, `git clone https://github.com/prysmaticlabs/geth-sharding ./go-ethereum`
-   Fork the our repository on Github: <https://github.com/prysmaticlabs/geth-sharding>
-   Add a remote to your fork
    \`git remote add YOURNAME <https://github.com/YOURNAME/geth-sharding>

Now you should have a remote pointing to the `origin` repo (geth-sharding) and to your forked, go-ethereum repo on Github. To commit changes and start a Pull Request, our workflow is as follows:

-   Create a new branch with a clear feature name such as `git checkout -b collations-pool`
-   Issue changes with clear commit messages
-   Run the linter and CI tester as follows `go run build/ci.go test && go run build/ci.go lint`
-   Push to your remote `git push YOURNAME collations-pool`
-   Go to the [geth-sharding](https://github.com/prysmaticlabs/geth-sharding) repository on Github and start a PR comparing `geth-sharding:master` with `go-ethereum:collations-pool` (your fork on your profile).
-   Add a clear PR title along with a description of what this PR encompasses, when it can be closed, and what you are currently working on. Github markdown checklists work great for this.

Pull requests must be cleanly rebased ontop of master. If master advances while your PR is in review, please keep rebasing it.
Before the pull request is merged, make sure that you squash your commits into one commit using `git rebase -i` and `git push -f`. After every commit the test suite must be passing.

## Contributor Responsibilities

We consider two types of contributions to our repo and categorize them as follows:

### Part-Time Contributors

Anyone can become a part-time contributor and help out on implementing sharding. The responsibilities of a part-time contributor include:

-   Engaging in Gitter conversations, asking the questions on how to begin contributing to the project
-   Opening up github issues to express interest in code to implement
-   Opening up PRs referencing any open issue in the repo. PRs should include:
    -   Detailed context of what would be required for merge
    -   Tests that are consistent with how other tests are written in our implementation
-   Proper labels, milestones, and projects (see other closed PRs for reference)
-   Follow up on open PRs
    -   Have an estimated timeframe to completion and let the core contributors know if a PR will take longer than expected

We do not expect all part-time contributors to be experts on all the latest sharding documentation, but all contributors should at least be familiarized with our sharding [README.md](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/README.md) and have gone through the required Ethereum readings as posted on our [READINGS.md](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/READINGS.md) document.

### Core Contributors

Core contributors are remote contractors of Prysmatic Labs, LLC. and are considered critical team members of our organization. Core devs have all of the responsibilities of part-time contributors plus the majority of the following:

-   Stay up to date on the latest sharding posts on ETHResearch
-   Monitor github issues and PR’s to make sure owner, labels, descriptions are correct
-   Formulate independent ideas, suggest new work to do, point out improvements to existing approaches
-   Participate in code review, ensure code quality is excellent, and have ensure high code coverage
-   Help with social media presence, write bi-weekly development update
-   Represent Prysmatic Labs at events to help spread the word on scalability research and solutions

We love working with people that are autonomous, bring independent thoughts to the team, and are excited for their work! We believe in a merit-based approach to becoming a core contributor, and any part-time contributor that puts in the time, work, and drive can become a core member of our team.

![eth](https://steemitimages.com/DQmV1NASyCJYusDjY1WCvpoWiXh32HyumQHFQhY8zYZ6WDH/source.gif)