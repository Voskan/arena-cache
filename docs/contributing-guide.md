# Contributing Guide for arena-cache

## Introduction

Thank you for considering contributing to `arena-cache`! We welcome contributions from the community to help improve and enhance the project. This guide provides instructions and best practices for contributing to `arena-cache`.

## Getting Started

### Prerequisites

- **Go 1.24 or later**: Ensure you have Go 1.24 or a later version installed, as `arena-cache` leverages the experimental `arena` allocator.
- **Git**: You will need Git for version control.

### Setting Up the Development Environment

1. **Fork the Repository**: Start by forking the `arena-cache` repository on GitHub to your own account.
2. **Clone the Repository**: Clone your forked repository to your local machine.
   ```bash
   git clone https://github.com/YOUR_USERNAME/arena-cache.git
   cd arena-cache
   ```
3. **Install Dependencies**: Use Go modules to install the necessary dependencies.
   ```bash
   go mod tidy
   ```
4. **Build the Project**: Compile the project to ensure everything is set up correctly.
   ```bash
   go build ./...
   ```

## Making Changes

### Coding Standards

- Follow the existing code style and conventions.
- Write clear, concise, and well-documented code.
- Ensure your code is covered by tests.

### Testing

- Run the test suite to ensure your changes do not break existing functionality.
  ```bash
  go test ./...
  ```
- Add new tests to cover any new functionality or changes.

## Submitting Changes

1. **Commit Your Changes**: Use clear and descriptive commit messages.
   ```bash
   git commit -m "Description of changes"
   ```
2. **Push to Your Fork**: Push your changes to your forked repository.
   ```bash
   git push origin your-branch-name
   ```
3. **Create a Pull Request**: Open a pull request on the original `arena-cache` repository.
   - Provide a clear description of your changes and the problem they solve.
   - Reference any related issues or pull requests.

## Code of Conduct

Please note that this project is governed by a [Code of Conduct](../CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Conclusion

We appreciate your contributions to `arena-cache`! If you have any questions or need assistance, feel free to open an issue or reach out to the maintainers.
