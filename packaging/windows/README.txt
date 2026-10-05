Jumpgate for Windows
====================

jumpgate-tray.exe   The desktop app. Double-click it: a window opens and a
                    Jumpgate icon appears in the notification area (right-click
                    it to quit). Needs the Microsoft Edge WebView2 Runtime,
                    which Windows 11 includes.

jumpgate.exe        The terminal. Double-click it for the terminal home, or run
                    it from a terminal: `jumpgate.exe help` lists the commands,
                    `jumpgate.exe open` opens the web app in your browser.

Both need Windows 10 version 1803, Windows Server 2019, or later.
Errors from the desktop app are written to %USERPROFILE%\.jumpgate\run\app.log.

These files are not code-signed yet, so SmartScreen may warn the first time:
choose "More info" and "Run anyway".
