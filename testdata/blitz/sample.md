# Sample Document

A markdown file for the `blitz` decoder, which lays it out with a real css
engine and hands back the rendered image.

## Emphasis

Plain text, *emphasis*, **strong emphasis**, and `inline code`.

## Lists

1. Ordered item
2. Ordered item
   - Nested unordered item
   - Nested unordered item
3. Ordered item

## Code

```go
func main() {
	fmt.Println("hello")
}
```

## Quote

> Rendering happens in one step: blitz parses the markdown, styles it, and
> paints the result headlessly -- no browser, no window.

## Table

| Decoder  | Input       | Output            |
|----------|-------------|-------------------|
| blitz    | `.md`       | image             |
| graphviz | `.dot`      | `image/svg+xml`   |
| mermaid  | `.mmd`      | `image/svg+xml`   |

---

See the [iv README](https://github.com/kenshaw/iv) for more.
