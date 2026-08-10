# Development

Use red/green TDD with the `github.com/stretchr/testify` library.

- Unit Tests: Use `./test.sh` to run tests.
- Lint: Run `./lint.sh` afterwards to ensure formatting is correct
- Manual testing: `./run.sh --qa-console`

# Commits

The writing style should be simple and close to what a human would want to
tell another human if they were curious about the change, or wanted to
keep there future self in the loop.

The format:

``` [<scope>] <short summary>

<1-3 sentences on the what, bullets only if a lot has changed>

<1-3 sentences on the why>
```

WRAP the lines at 72 characters for the body and keep the first line
at under 50 if possible.
