---
title: Event Graph
description: Event and on_action neighborhood from the live session harvest.
weight: 4
---

# Event Graph

Browse what an event fires and what fires it, using the **live** workspace harvest (rebuilt on open).

{{< shot caption="Graph with origin groups + inspector." >}}

## How to use it

1. Pick a **root** event. Namespace narrows the picker; leave it empty to search every id.
2. **Origins** filter which mods (and game files) feed the graph.
3. **Double-click** a node to re-root.
4. **Drag** is temporary until you change layout or root.
5. **Recenter** only pans to fit — it does not undo a layout change.

Layout is layered (dagre) with origin groups (Vue Flow parents). Via-effect edges (up to a few hops) stay so a `trigger_event` inside a scripted effect still draws.

The inspector templates the payload: title, desc, gates, effects, loc rows.

## From the IDE

Hover an event id in the [IDE]({{< relref "/pages/ide" >}}) (definition or `trigger_event`) and choose **Open in Event Graph**.
