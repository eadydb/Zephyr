# STYLE GUIDE

## Go Coding Standards
- Follow standard Go formatting (gofmt)
- Use meaningful variable and function names
- Keep functions under 200 lines, preferably under 100
- Keep files under 500 lines

## Project Structure Guidelines
- Use internal/ for private packages
- Use pkg/ for public APIs
- Organize by functionality, not by type
- Keep plugin interfaces clean and minimal

## Documentation Standards
- Use Go doc comments for public APIs
- Provide examples in examples/ directory
- Keep README files up to date
- Document plugin interfaces thoroughly

## Code Quality Principles
- Simplicity over complexity
- Clear error handling
- Comprehensive testing
- Consistent naming conventions

## Plugin Development Guidelines
- Implement required interfaces
- Provide clear plugin metadata
- Include usage examples
- Follow plugin lifecycle patterns 