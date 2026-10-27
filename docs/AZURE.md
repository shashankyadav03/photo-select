# Hosting the friends homepage on Azure

The homepage in `site/` is a React (Vite) page that friends open to install Photo Select.
It detects their computer, gives them a one-line installer (or a zip), and walks them through
choosing a photo folder and swiping. The app zips are built from this repo and hosted on the
same site, so friends never need Git, Go or GitHub.

It runs on **Azure Static Web Apps, Free plan** (no cost, HTTPS included). The site is a
download page only; photos never go to Azure.

## One-time setup (about 5 minutes)

### 1. Create the Static Web App

**Azure Portal:** Create a resource → **Static Web App** →
- Resource group: new, e.g. `photo-select`
- Name: e.g. `photo-select`
- Plan type: **Free**
- Deployment source: **Other** (this repo already has its own workflow)
- Create.

**Or Azure CLI:**

```bash
az group create -n photo-select -l eastus2
az staticwebapp create -n photo-select -g photo-select -l eastus2 --sku Free
```

### 2. Give GitHub the deployment token

Portal: open the Static Web App → **Overview → Manage deployment token** → copy.
CLI: `az staticwebapp secrets list -n photo-select -g photo-select --query properties.apiKey -o tsv`

In GitHub: repo **Settings → Secrets and variables → Actions → New repository secret**
- Name: `AZURE_STATIC_WEB_APPS_API_TOKEN`
- Value: the token

Never paste the token into an issue, chat or commit.

### 3. Deploy

Every push to `main` deploys automatically (`.github/workflows/site.yml`). To deploy now:
GitHub → **Actions → Deploy friends homepage (Azure) → Run workflow**.

### 4. Share the link

The address is on the Static Web App's **Overview** page (**URL**), like
`https://happy-sea-0a1b2c3d4.azurestaticapps.net`. Send that to your friends.

Optional: **Custom domains** in the Static Web App lets you use your own domain (free HTTPS).

## What friends see

1. **Install:** one copy-paste command for Mac/Linux (Terminal) or Windows (PowerShell), or a zip download.
   The command downloads the right build from the site, installs it (`~/PhotoSelect` or
   `%LOCALAPPDATA%\PhotoSelect`), adds a Desktop launcher and starts the app.
   Files fetched this way aren't quarantined, so there are no Gatekeeper/SmartScreen warnings.
2. **Choose folders:** the app opens its setup page at `http://localhost:8787`; they paste or pick their photo folder.
3. **Swipe:** they open the address it shows on their phone, on the same Wi-Fi.

## Build it locally

```bash
scripts/build-site.sh          # builds the app zips + site into site/dist/
cd site && npx vite preview    # http://localhost:4173
```

For editing just the page: `cd site && npm install && npm run dev` (downloads show as unavailable
until `scripts/build-site.sh` has run once).
