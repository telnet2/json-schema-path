# Contributing to json-schema-path

Thank you for your interest in contributing to json-schema-path! We welcome contributions from the community.

## How to Contribute

### Reporting Issues

If you find a bug or have a suggestion for improvement:

1. Check if the issue already exists in [GitHub Issues](https://github.com/telnet2/json-schema-path/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Your environment (Go version, OS, etc.)

### Submitting Changes

1. **Fork the repository**
   ```bash
   git clone https://github.com/telnet2/json-schema-path.git
   cd json-schema-path
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **Make your changes**
   - Write clear, idiomatic Go code
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed

4. **Test your changes**
   ```bash
   # Run all tests
   go test ./...

   # Run with coverage
   go test -cover ./...

   # Run benchmarks
   go test -bench=. ./...

   # Test specific components
   go test ./parser -v
   go test ./json -v
   go test ./tree -v
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "Add amazing feature"
   ```

   Write clear commit messages that describe what changed and why.

6. **Push to your fork**
   ```bash
   git push origin feature/amazing-feature
   ```

7. **Submit a Pull Request**
   - Go to the original repository on GitHub
   - Click "New Pull Request"
   - Select your feature branch
   - Provide a clear description of your changes
   - Link any related issues

## Development Guidelines

### Code Style

- Follow standard Go formatting (use `gofmt` or `goimports`)
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions focused and reasonably sized

### Testing

- Write unit tests for new functionality
- Maintain or improve code coverage
- Add benchmarks for performance-critical code
- Test edge cases and error conditions

### Documentation

- Update README.md if adding user-facing features
- Add godoc comments for exported functions and types
- Update relevant documentation in the `docs/` folder
- Include examples for new features

### Performance

- Run benchmarks before and after changes
- Avoid unnecessary allocations
- Profile performance-critical paths
- Document any performance trade-offs

## Project Structure

```
json-schema-path/
├── cmd/schemapath/         # CLI application
├── docs/                   # Documentation files
├── json/                   # JSON processing with sonic/AST
├── parser/                 # Expression parser & lexer
├── spec/                   # Grammar specification & AST nodes
├── tree/                   # Pattern matching trie implementation
└── validators/             # Validator implementations
```

## Questions?

- Open a [GitHub Discussion](https://github.com/telnet2/json-schema-path/discussions)
- Check existing issues and pull requests
- Read the documentation in `docs/` folder

## Code of Conduct

- Be respectful and constructive
- Welcome newcomers and help them learn
- Focus on what is best for the community
- Show empathy towards other community members

## License

By contributing to json-schema-path, you agree that your contributions will be licensed under the MIT License.
