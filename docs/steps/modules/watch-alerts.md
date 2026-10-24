---
id: watch-alerts
title: See and send health-watch alerts
summary: Where health-watch alerts show, and how to have the box post them to a webhook such as ntfy.
recap: |
  Open alerts show on the machine card, on the TUI fleet row (!N) and as an amber tray dot.
  The box keeps watching with the app closed. For alerts on your phone, give the watch an https webhook such as an ntfy topic URL.
requires: [watch-run]
verified: {}
---

The health watch keeps its alerts on the box, in its state file. The box's
gate reads that file when asked (`modules.read`). For a box reached over SSH
only, the app reads it over SSH.

## Where alerts show

While the app runs, it reads every box's alerts as part of the fleet poll:

- **App.** The machine's card shows a badge such as **2 alerts** and the worst
  alert's message.
- **TUI.** The box's row in the fleet view shows `!2` after its name.
- **Tray.** The status dot turns amber, and its tooltip says, for example,
  `Jumpgate — 2 alerts on box-1`. A box that stops answering turns it amber
  too. A blocked gateway still turns it red.

The fleet API (`GET /api/fleet`) carries each row's `alertCount`,
`alertMessage` and `alerts` (key, message, since).

## When the app is closed

The box keeps watching. It keeps the last 50 cleared alerts too, so you can
see what happened when you come back.

To hear about alerts while the app is closed, give the watch a **webhook**.
The box then posts each new alert, and each one that clears, as JSON:

```json
{"box":"box-1","key":"disk-90","message":"Disk is 93% full.","since":"2026-10-23T12:00:00Z","cleared":null}
```

- It is off unless you set it.
- It must be `https://`. Redirects are not followed.
- A slow or failing webhook never delays the watch. It is tried twice, and a
  failure is logged without the URL.

### Use ntfy

1. Pick a topic name that is hard to guess. Anyone who knows it can read it.
2. In the app, open the machine, and under **Health watch** paste
   `https://ntfy.sh/<your-topic>` into **Webhook**. Click **Save watch
   settings**.
3. Subscribe to the topic in the ntfy app on your phone.

The app never shows the saved webhook again. Leave the field blank to keep
it, or tick **Remove the saved webhook** to turn it off.
