---
title: Display
description: Header popover — themes, scale, fonts, visible tools, Guide toast.
weight: 9
---

# Display

The monitor button in the **header**. These prefs survive **Reset all data**.

{{< shot caption="Display popover (theme + visible tools)." >}}

## Theme (header, next to Display)

The family menu and light/dark toggle sit in the header beside this popover. Families: **PMT**, **CK3**, **EU5**, **Vic3**, **Catppuccin**, **One**, **GitHub**, **Horizon**. Same palette as the IDE workbench.

## Sliders

| Control | What it changes |
|---------|-----------------|
| **UI scale** | App chrome (%) |
| **Editor font** | Script / loc editor |
| **Workbench UI** | Explorer, Problems, panels, menus (root folders are 2px larger) |

## Visible tools

Which toolbar pills show: IDE, Event Graph, Workspace Health, Patch Center, Publish. Hidden tools are still in the router; Library just will not land on them.

## Guide toast

Snooze or turn off the toast that points at the [Guide]({{< relref "/pages/ide" >}}) pull-tab.

## Reload UI

Remounts chrome and the workbench webview after font/theme changes that need a clean sheet.
