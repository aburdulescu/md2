These are all the Markdown features supported
(CommonMark + Github extensions + some other extensions).

Run this file through the tool to see the resulting HTML:

`md2 example.md > example.html`

# Heading

# h1
## h2
### h3

# Bold

**this is bold**

# Italic

*this is italic*

# Blockquote

> this is a blockquote

# Horizontal rule

---

# List

## Ordered

1. a
2. b

## Unordered

- a
- b

# Code

`this is code`

# Link

[description](target)

# Image

![alt text](image.jpg)

# Autolink

www.commonmark.org

# Fenced code

```
multi line
code
block
```

# Table

|foo|bar|
|---|---|
|aaaaaaa|xxxx|

# Footnote

That's some text with a footnote.[^1]

[^1]: And that's the footnote.

    That's the second paragraph.

# Heading ID

### My Great Heading {#custom-id}

# Definition list

Apple
:   Pomaceous fruit of plants of the genus Malus in
the family Rosaceae.

Orange
:   The fruit of an evergreen tree of the genus Citrus.

# Strikethrough

~~The world is flat.~~

# Task List

- [x] foo
  - [ ] bar
  - [x] baz
- [ ] bim
