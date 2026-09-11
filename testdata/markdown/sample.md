# Sample Document

A markdown file for the `markdown` decoder, which renders it to a pdf and
hands that back to the decoding pipeline.

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

> Rendering happens in two steps: goldmark converts the markdown to a pdf,
> then libvips rasterizes a page of it.

## Table

| Decoder  | Input       | Output            |
|----------|-------------|-------------------|
| markdown | `.md`       | `application/pdf` |
| graphviz | `.dot`      | `image/svg+xml`   |
| mermaid  | `.mmd`      | `image/svg+xml`   |

---

See the [iv README](https://github.com/kenshaw/iv) for more.
