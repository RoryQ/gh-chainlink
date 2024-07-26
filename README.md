# Chain-link

Github `gh` cli extension to help link chained PRs together.

### Installation

To install simply run

```
gh extension install roryq/gh-chainlink
```

And to update

```
gh extension upgrade roryq/gh-chainlink
```

### Usage
#### Example workflow
The expected workflow is to have a parent issue that has a list or links to related PRs (or issues).
The list can be numbered, bulleted or a checklist.
The body of each list should be a link to a github issue, or a shorthand issue reference for issues in the same repo.

e.g. Pull Request `#100` has an issue description like

```markdown
# Description
Here is my first PR for the issue.

## PR Chain
<!--chainlink-->
1. #100
2. #101
3. #102
```

The related PRs `#101` and `#102` only have a description but no links to the previous chained PRs.
```markdown
# Description
Here is my second PR for the issue.
```
```markdown
# Description
Here is my third PR for the issue.
```
#### Run the command
Running the following command from the context of your repo will synchronise the PR chain across each of the linked PRs

```
gh chainlink 100
```

Alternatively you can provide a full url

```
gh chainlink "github.com/roryq/gh-chainlink/pull/100"
```

#### See the results
The PRs will be updated like so

```markdown
# Description
Here is my first PR for the issue.

## PR Chain
<!--chainlink-->
1. #100 ← you are here
2. #101
3. #102
```

```markdown
# Description
Here is my second PR for the issue.

## PR Chain
<!--chainlink-->
1. #100
2. #101 ← you are here
3. #102
```

```markdown
# Description
Here is my third PR for the issue.

## PR Chain
<!--chainlink-->
1. #100 
2. #101
3. #102 ← you are here
```

If at any point you add another PR. You can update one of the issues and run the command again to propagate the changes.