# Makefiles

## TLDR

Common Makefiles for Fleet projects.

See [fleet/cicd/makefiles-example](https://gitlab.agodadev.io/fleet/cicd/makefiles-example) for an example project.

## Usage

### Setup

Add this repository as a subtree in directory `makefiles`:

```sh
git remote add -f makefiles git@gitlab.agodadev.io:fleet/cicd/makefiles.git
git subtree add --prefix makefiles makefiles main --squash
```

Create `Makefile` in the root of your project:

```
include makefiles/go.mk

# Add your targets here
e2e-test:
    scripts/e2e-test.sh
```

### Update

```sh
git subtree pull -q --prefix makefiles makefiles main --squash -m "ci: update makefiles"
```

## Targets

Run `make help` to see all available targets.

```
help                           Help
generate                       Generate
format                         Format
lint                           Lint
test                           Run unit tests
integration-test               Run integration tests
e2e-test                       Run e2e tests
coverage                       Collect coverage
```

