# Rich Text Editor

A rich text editor example: a multiline text input with a toolbar that
applies ranged style overrides interactively, wrapped in the shared editor
chrome (menu bar, status bar, and file dialogs).

* Bold, italic, underline, and strikethrough toggles (also in the Format
  menu). A button lights up when the property is uniformly on across the
  selection. With no selection, a toggle applies to the text typed next.
* Text color and highlight palettes with a default entry that restores the
  base style.
* Font size stepping through a fixed scale ladder.
* Clearing the style overrides of the selection.
* Undo and redo, restoring the styles alongside the text.
* New and Open with an unsaved-changes confirmation. Saving is not
  implemented yet: a saved file would lose its styles, as a rich text file
  format is still to be decided, so the save menu items are disabled.
* Text search (Cmd/Ctrl+F).

# Licenses

## `InterVariable-Italic.ttf.gz`

Inter

* https://github.com/rsms/inter

Adopted `InterVariable-Italic.ttf` from the Inter 4.1 release, gzipped. It
complements the upright Inter face bundled in `basicwidget`.

Copyright (c) 2016 The Inter Project Authors (https://github.com/rsms/inter)

```
SIL OPEN FONT LICENSE Version 1.1
```

## The images in `resource`

Material Symbols & Icons - Google Fonts

* https://fonts.google.com/icons
* https://github.com/google/material-design-icons

```
Apache License, Version 2.0
```
