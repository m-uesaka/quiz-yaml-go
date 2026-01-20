# Instructions for GitHub Copilot

## Basic requirements

- Ensure your repository has a clear README file that outlines the purpose and usage of the project.
- Ask a question to the user if it is necessary to clarify the context or requirements before providing code suggestions.
- Divide complex tasks into smaller, manageable parts and proceed step-by-step with `git commit` after completing each part.

## document structure

- `docs/schema.md` contains the database schema and relationships.
- `docs/requirements.md` lists the functional and non-functional requirements.
- `docs/reports/*.md` contains various reports related to the project.

## Development guidelines

- Refer to [`docs/requirements.md`](/docs/requirements.md) to understand the requirements before writing any code.
- If you are instructed to write "plan", write a detailed tasks in `docs/tasks.md` before starting the implementation.
  - You may delete the existing `docs/tasks.md` file if it exists.
  - You should not start coding until the user approves the plan.
- If you are instructed to impelemnt a feature, write the code by following `docs/tasks.md`.
  - Add appropriate tests for the implemented feature.

## git commit guidelines

- You should create semantic commit messages.
- Put the following prefix to commit message.
  - "chore" : "Build process or auxiliary tool changes"
  - "ci" : "CI related changes"
  - "data": "Changes on data"
  - "docs" : "Documentation changes"
  - "feat" : "A new feature"
  - "fix" : "A bug fix"
  - "perf" : "A code change that improves performance"
  - "refactor" : "A code change that neither fixes a bug or adds a feature"
  - "style" : "Markup, white-space, formatting, missing semi-colons..."
  - "test" : "Adding missing tests"
  - Example of commit message: "refactor: simplify class"

## test guidelines

- Follow the Arrange, Act, and Assert (AAA) pattern ([Khorikov, 2022, Section 3.1](#references)).
  - In the Arrange section, you bring the system under test (SUT) and its dependencies to a desired state.
  - In the Act section, you call methods on the SUT, pass the prepared dependencies, and capture the output value (if any).
  - In the Assert section, you verify the outcome. The outcome may be represented by the return value, the final state of the SUT and its collaborators, or the methods the SUT called on those collaborators.
- - Divide normal and invalid cases as different test methods in tests. ([Khorikov, 2020, Section 3.5](#references))
- Use parametrize for different Arrange cases as possible.
  - However, we avoid parametrize and divide the test method for readability and maintainability in the following case.
    - when test code contains if-else statement due to parametrization. ([Khorikov, 2020, Section 3.1.3](#references))
    - when parametrization may mix both normal and invalid cases. ([Khorikov, 2020, Section 3.5](#references))

## Task runner guidelines

- Use [Taskfile](https://taskfile.dev/) to automate common tasks such as
  - setting up the development environment,
  - running tests, and
  - deploying the application, and so on.

## References

- [Khorikov, V. (2020), Unit Testing Principles, Practices, and Patterns: Effective testing styles, patterns, and reliable automation for unit testing, mocking, and integration testing with examples in C#. Manning.](https://www.notion.so/datalabs-jp/Unit-Testing-Principles-Practices-and-Patterns-Effective-testing-styles-patterns-and-reliable-a-c32c418bff7147d1b5d39dbaa8d48b5b)  
