# 📸 Photo Select

Go through a huge photo library one photo at a time, Tinder-style, and keep only the best ones.

- **Swipe right** to keep. The photo is copied into a `Selected` folder.
- **Swipe left** to skip. The photo stays where it is and is marked as rejected.
- **Swipe up** for *maybe*. You see it again after you've been through everything else.
- **↶** undoes the last swipe, including the copy.

It runs on the computer that has the photos (for example a MacBook with an external drive). You swipe on your phone over Wi-Fi, and you can mirror the phone to a TV so everyone can vote. Progress is saved after every swipe, so you can stop and carry on later. It was built for picking about 500 photos out of 25,000.

## Quick start (Mac)

1. Download the zip for your Mac from the [Releases page](../../releases): **mac-apple-silicon** for M1/M2/M3/M4 Macs (MacBook Air from 2020 on), **mac-intel** for older Macs. Unzip it.
2. Plug in the drive with your photos.
3. **Right-click** `Start Photo Select.command` → **Open** → **Open**. You only need the right-click the first time, because the app isn't from the App Store. A Terminal window opens. Keep it open while you swipe.
4. A setup page opens in your browser. Choose:
   - the folder with all your photos (sub-folders are included)
   - where the selected photos should go (a new folder next to it is suggested)
   - your goal, e.g. 500
5. Click **Start swiping**. The page shows an address like `http://192.168.1.23:8787`.
6. On your iPhone, on the **same Wi-Fi**, open that address in Safari.
   - Tip: Share → **Add to Home Screen** gives you a full-screen app.
7. To vote on the TV: iPhone Control Centre → **Screen Mirroring** → pick your TV (AirPlay TV or Apple TV). Turning the phone sideways fills more of the TV.

To quit, close the Terminal window. Next time, open the app again and it carries on where you left off.

> **"Can't be opened because Apple cannot check it for malicious software"?**
> Right-click the file → **Open** instead of double-clicking. Or open Terminal in the unzipped folder and run `xattr -dr com.apple.quarantine .` once.

## Controls

| Action | Phone | Keyboard (on the computer) |
|---|---|---|
| Keep | swipe right / ♥ | → |
| Skip | swipe left / ✕ | ← |
| Maybe, show again later | swipe up / ? | ↑ |
| Undo last | ↶ | Z or Backspace |

Several phones can be open at the same time. They stay in sync.

## What happens to my files?

| Setting | Default | Options |
|---|---|---|
| When you swipe right | **Copy** into `<output>/Selected/` | Move instead (saves space, and undo moves the photo back) |
| When you swipe left | **Leave it where it is** | Copy or move into `<output>/Rejected/` |

- Your originals are never changed or deleted unless you choose **Move**.
- All selected photos go into one folder. If two photos have the same name (cameras reuse `IMG_0001.JPG`), the second one becomes `IMG_0001 (2).JPG`.
- Progress is saved in `<output>/.photo-select/decisions.jsonl` after every swipe. It travels with the drive, so you can carry on from another computer by choosing the same folders.
- **Download decisions (CSV)** on the setup page gives you a spreadsheet of every photo and what you decided.
- To start over completely, quit the app and delete the `.photo-select` folder inside your output folder.

## Supported photos

JPEG, PNG, HEIC/HEIF (iPhone photos), GIF, WebP, TIFF, BMP, and camera RAW files (DNG, CR2, CR3, NEF, ARW, RAF, ORF, RW2, SRW, PEF). Videos are skipped.

On a Mac, large photos, HEIC and RAW are shrunk to phone size on the fly with macOS's built-in `sips` tool, so swiping stays quick over Wi-Fi. On Windows or Linux the original file is sent. That works for JPEG, PNG and (on an iPhone) HEIC, but RAW files show "Preview not available". You can still swipe them.

## Troubleshooting

- **Phone can't open the address:** make sure both devices are on the same Wi-Fi, not a guest network. The first time you run the app, macOS may ask whether to allow incoming connections. Click **Allow**. If you clicked Deny, fix it in System Settings → Network → Firewall → Options.
- **"Photo folder not found":** the drive isn't plugged in, or it was renamed. Plug it in and click **Start swiping** again.
- **Port already in use:** the app picks the next free port automatically. Use the address it prints.

## Command-line options

```
./photo-select [-port 8787] [-no-browser] [-version]
```

## For developers

Written in Go with no dependencies outside the standard library. The web UI in `web/` is embedded in the binary.

```
go run .                 # run locally
go test -race ./...      # tests
scripts/build.sh v1.0.0  # zips for Mac (Apple Silicon + Intel), Windows, Linux into dist/
```

Pushing a tag like `v1.0.0` runs CI and attaches the zips to a GitHub Release.
