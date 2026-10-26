# 📸 Photo Select

Go through a huge photo library one photo at a time, Tinder-style, and keep only the best ones.

- **Swipe right** to keep. The photo is copied into a `Selected` folder.
- **Swipe left** to skip. The photo stays where it is and is marked as rejected.
- **Swipe up** for *maybe*. You see it again after you've been through everything else.
- **↶** undoes the last swipe, including the copy.

It runs on the computer that has the photos (a Mac, Windows or Linux machine, for example a MacBook with an external drive). You swipe on your phone over Wi-Fi, and you can mirror the phone to a TV so everyone can vote. Progress is saved after every swipe, so you can stop and carry on later. It was built for picking about 500 photos out of 25,000.

## Get it

**Easiest: the homepage.** If someone sent you a Photo Select link, open it on your computer and follow the three steps there. It installs the app with one copy-paste command. (Hosting that page yourself: see [docs/AZURE.md](docs/AZURE.md).)

Or pick **one** of these.

**A. Download the app (easiest, no coding).** Open the [Releases page](../../releases) and download the zip for your computer:
**mac-apple-silicon** (M1/M2/M3/M4 Macs, MacBook Air from 2020 on), **mac-intel** (older Macs), **windows**, or **linux**. Unzip it.

**B. Run it from this repo.** You need [Go](https://go.dev/dl/) 1.22 or newer, installed once.

```bash
git clone https://github.com/shashankyadav03/photo-select.git
```

```bash
cd photo-select && go run .
```

**C. Make your own copy.** Click **Fork** on GitHub, then use option B with your fork's URL. To publish your own downloadable zips, push a tag (`git tag v1.0.0 && git push origin v1.0.0`) and GitHub Actions attaches them to a Release on your fork.

## Your photos stay on your computer

You don't put photos into this repo. Keep them wherever they already are (an external drive, `~/Pictures`, a folder on the desktop) and choose that folder on the setup page. The app reads that folder in place and writes its picks to the output folder you choose. Nothing is uploaded anywhere. Phones on your Wi-Fi only see the photos while the app is running.

Any folder layout works. Sub-folders are included, and photos are shown oldest first by the date the camera recorded. Picks keep their sub-folder, so `Trips/Goa/IMG_1.JPG` ends up at `Selected/Trips/Goa/IMG_1.JPG`:

```
My Photos/                 <- choose this as the photo folder
  2023/ ...
  Trips/Goa/IMG_1.JPG
My Photos - Selected/      <- suggested output folder (created for you)
  Selected/Trips/Goa/IMG_1.JPG
  .photo-select/           <- saved progress, keep it to carry on later
```

If you cloned the repo, **don't choose folders inside the repo folder**. `.gitignore` blocks photos and progress files anyway, just in case.

## Start swiping

1. Plug in the drive with your photos, if they're on one.
2. Start the app:
   - **Mac download:** **right-click** `Start Photo Select.command` → **Open** → **Open**. You only need the right-click the first time, because the app isn't from the App Store. A Terminal window opens. Keep it open while you swipe.
   - **Windows download:** double-click `photo-select.exe`. If SmartScreen warns you, click **More info** → **Run anyway**.
   - **From the repo:** `go run .`
3. A setup page opens in your browser (or go to http://localhost:8787). Choose:
   - the folder with all your photos
   - where the selected photos should go (a new folder next to it is suggested)
   - your goal, e.g. 500
4. Click **Start swiping**. The page shows an address like `http://192.168.1.23:8787`.
5. On your phone, on the **same Wi-Fi**, open that address in the browser.
   - Tip on iPhone: Share → **Add to Home Screen** gives you a full-screen app.
6. To vote on the TV: iPhone Control Centre → **Screen Mirroring** → pick your TV (AirPlay TV or Apple TV). Turning the phone sideways fills more of the TV.

To quit, close the Terminal window (or press Ctrl+C). Next time, start the app again and choose the same folders. It carries on where you left off.

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
scripts/build-site.sh    # the friends homepage (site/, React) with those zips bundled in
```

The homepage in `site/` deploys to Azure Static Web Apps on every push to `main`; see [docs/AZURE.md](docs/AZURE.md).

Pushing a tag like `v1.0.0` runs CI and attaches the zips to a GitHub Release.

Pull requests are welcome. Run `gofmt -l . && go vet ./... && go test -race ./...` before opening one.
