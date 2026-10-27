import { useEffect, useState } from 'react'
import { PLATFORMS, detectMacChip, detectOS } from './platform.js'
import { CopyCommand, DownloadButton, Kbd, Note, Steps, Tabs } from './components.jsx'

const LOCAL_URL = 'http://localhost:8787'
const site = typeof window !== 'undefined' ? window.location.origin : ''

export default function App() {
  const [detected] = useState(detectOS)
  const [os, setOS] = useState(detected === 'phone' ? 'mac' : detected)
  const [chip, setChip] = useState(null)
  const [manifest, setManifest] = useState(null)

  useEffect(() => {
    if (detected === 'mac') detectMacChip().then(setChip)
    fetch('/downloads/manifest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then(setManifest)
      .catch(() => setManifest(null))
  }, [detected])

  return (
    <>
      <Header />
      <main>
        {detected === 'phone' && <PhoneBanner />}
        <Hero os={os} />
        <HowItWorks />
        <Install os={os} setOS={setOS} chip={chip} manifest={manifest} />
        <ChooseFolders os={os} setOS={setOS} />
        <Swipe />
        <Files />
        <Help os={os} />
      </main>
      <Footer manifest={manifest} />
    </>
  )
}

function Header() {
  return (
    <header className="header">
      <a href="#top" className="brand">
        <img src="/favicon.svg" alt="" width="28" height="28" /> Photo Select
      </a>
      <nav>
        <a href="#install">Install</a>
        <a href="#folders">Folders</a>
        <a href="#swipe">Swipe</a>
        <a href="#help">Help</a>
      </nav>
    </header>
  )
}

function PhoneBanner() {
  return (
    <div className="phone-banner">
      <b>📱 You're on a phone.</b> Photo Select is installed on the <b>computer</b> that has your photos.
      Open this page on that computer first. You'll use your phone later to swipe.
      <CopyCommand command={site} label="Copy link" />
    </div>
  )
}

function Hero({ os }) {
  return (
    <section className="hero" id="top">
      <div className="hero-text">
        <h1>
          Too many photos?
          <br />
          <span className="grad">Swipe to keep the best.</span>
        </h1>
        <p className="lead">
          Photo Select shows your photos one at a time on your phone. Swipe right to keep, left to skip.
          Go from 25,000 photos to your favourite 500, with friends and family voting on the TV.
        </p>
        <div className="hero-actions">
          <a className="button primary big" href="#install">
            Install on {PLATFORMS[os]} →
          </a>
          <a className="button big" href="#how">
            How it works
          </a>
        </div>
        <ul className="badges">
          <li>🔒 Photos never leave your computer</li>
          <li>💸 Free</li>
          <li>⏱ Set up in 2 minutes</li>
          <li>💾 Progress saved after every swipe</li>
        </ul>
      </div>
      <PhoneMock />
    </section>
  )
}

function PhoneMock() {
  return (
    <div className="phone" aria-hidden="true">
      <div className="phone-stats">
        <span className="s-yes">♥ 212 / 500</span>
        <span className="s-no">✕ 3,904</span>
        <span className="s-maybe">? 41</span>
      </div>
      <div className="phone-stage">
        <div className="phone-card back" />
        <div className="phone-card front">
          <div className="sun" />
          <div className="hill h1" />
          <div className="hill h2" />
          <span className="stamp">KEEP</span>
        </div>
      </div>
      <div className="phone-buttons">
        <span className="pb no">✕</span>
        <span className="pb maybe">?</span>
        <span className="pb yes">♥</span>
      </div>
    </div>
  )
}

function HowItWorks() {
  const cards = [
    ['💻', 'Install the app', 'One command or one download on the computer that has your photos. No account, no sign-up.'],
    ['📁', 'Choose your photo folder', 'Point it at the folder (or drive) with your photos and pick where the keepers should go.'],
    ['📱', 'Swipe on your phone', 'Open the address it shows on any phone on the same Wi-Fi. Mirror to the TV so everyone can vote.'],
  ]
  return (
    <section id="how" className="section">
      <h2>How it works</h2>
      <div className="cards">
        {cards.map(([icon, title, text], i) => (
          <div className="card" key={title}>
            <div className="card-num">{i + 1}</div>
            <div className="card-icon">{icon}</div>
            <h3>{title}</h3>
            <p>{text}</p>
          </div>
        ))}
      </div>
    </section>
  )
}

function Install({ os, setOS, chip, manifest }) {
  return (
    <section id="install" className="section">
      <h2>
        <span className="step-badge">1</span> Install Photo Select
      </h2>
      <p className="section-lead">
        Do this on the computer that has your photos. Pick your computer below (we guessed for you).
      </p>
      <Tabs tabs={PLATFORMS} value={os} onChange={setOS} />
      <div className="panel">
        {os === 'mac' && <InstallMac chip={chip} manifest={manifest} />}
        {os === 'windows' && <InstallWindows manifest={manifest} />}
        {os === 'linux' && <InstallLinux manifest={manifest} />}
      </div>
    </section>
  )
}

function InstallMac({ chip, manifest }) {
  return (
    <>
      <h3>
        Easiest: paste one line <span className="pill">Recommended</span>
      </h3>
      <Steps>
        <li>
          Open <b>Terminal</b>: press <Kbd>⌘</Kbd> + <Kbd>Space</Kbd>, type <b>Terminal</b>, press <Kbd>Return</Kbd>.
        </li>
        <li>
          Click <b>Copy</b>, paste into Terminal (<Kbd>⌘</Kbd> + <Kbd>V</Kbd>) and press <Kbd>Return</Kbd>:
          <CopyCommand command={`curl -fsSL ${site}/install.sh | bash -s ${site}`} />
        </li>
        <li>
          It downloads the app, puts a <b>Photo Select</b> icon on your Desktop and starts it. Your browser opens the
          setup page by itself. <b>Keep the Terminal window open</b> while you swipe.
        </li>
        <li>
          If your Mac asks <i>“Do you want the application photo-select to accept incoming network connections?”</i>{' '}
          click <b>Allow</b>. That's how your phone connects.
        </li>
      </Steps>
      <Note>
        <b>Next time:</b> double-click <b>Photo Select</b> on your Desktop.
      </Note>

      <details className="alt">
        <summary>Or download the app instead</summary>
        <div className="downloads">
          <DownloadButton manifest={manifest} label="mac-apple-silicon" primary={chip !== 'intel'}>
            Mac with Apple chip (M1–M4)
          </DownloadButton>
          <DownloadButton manifest={manifest} label="mac-intel" primary={chip === 'intel'}>
            Mac with Intel chip
          </DownloadButton>
        </div>
        <p className="muted small">
          {chip
            ? `This looks like a Mac with ${chip === 'intel' ? 'an Intel' : 'an Apple'} chip.`
            : 'Not sure which one?'}{' '}
          Click the Apple menu → <b>About This Mac</b>: <i>Chip: Apple M…</i> means Apple chip, <i>Processor: Intel</i> means
          Intel. Most Macs from 2021 on have an Apple chip.
        </p>
        <Steps>
          <li>Open the downloaded zip (it unzips by itself in Downloads).</li>
          <li>
            <b>Right-click</b> <code>Start Photo Select.command</code> → <b>Open</b> → <b>Open</b>.
          </li>
          <li>
            If it says it <i>can't be opened</i>: open <b>System Settings → Privacy &amp; Security</b>, scroll down and
            click <b>Open Anyway</b> next to “Start Photo Select”. You only do this once. (It's blocked because the
            app isn't from the App Store.)
          </li>
        </Steps>
      </details>
    </>
  )
}

function InstallWindows({ manifest }) {
  return (
    <>
      <h3>
        Easiest: paste one line <span className="pill">Recommended</span>
      </h3>
      <Steps>
        <li>
          Open <b>PowerShell</b>: press the <Kbd>⊞ Win</Kbd> key, type <b>PowerShell</b>, press <Kbd>Enter</Kbd>.
        </li>
        <li>
          Click <b>Copy</b>, paste into PowerShell (right-click or <Kbd>Ctrl</Kbd> + <Kbd>V</Kbd>) and press{' '}
          <Kbd>Enter</Kbd>:
          <CopyCommand command={`$u='${site}'; irm $u/install.ps1 | iex`} />
        </li>
        <li>
          It downloads the app, puts a <b>Photo Select</b> shortcut on your Desktop and starts it. A black window opens
          and your browser shows the setup page. <b>Keep the black window open</b> while you swipe.
        </li>
        <li>
          If <b>Windows Firewall</b> asks, tick <b>Private networks</b> and click <b>Allow access</b>. That's how your
          phone connects.
        </li>
      </Steps>
      <Note>
        <b>Next time:</b> double-click <b>Photo Select</b> on your Desktop.
      </Note>

      <details className="alt">
        <summary>Or download the app instead</summary>
        <div className="downloads">
          <DownloadButton manifest={manifest} label="windows" primary>
            Windows (64-bit)
          </DownloadButton>
        </div>
        <Steps>
          <li>
            Right-click the downloaded zip → <b>Extract All…</b> → <b>Extract</b>.
          </li>
          <li>
            Double-click <code>photo-select.exe</code> in the extracted folder.
          </li>
          <li>
            If a blue <i>“Windows protected your PC”</i> box appears, click <b>More info</b> → <b>Run anyway</b>. (It
            appears because the app isn't from the Microsoft Store.)
          </li>
        </Steps>
      </details>
    </>
  )
}

function InstallLinux({ manifest }) {
  return (
    <>
      <h3>
        Easiest: paste one line <span className="pill">Recommended</span>
      </h3>
      <Steps>
        <li>Open a terminal.</li>
        <li>
          Paste this and press <Kbd>Enter</Kbd>:
          <CopyCommand command={`curl -fsSL ${site}/install.sh | bash -s ${site}`} />
        </li>
        <li>
          It installs into <code>~/PhotoSelect</code> and starts the app. Next time run{' '}
          <code>~/PhotoSelect/photo-select</code>.
        </li>
      </Steps>
      <details className="alt">
        <summary>Or download the app instead</summary>
        <div className="downloads">
          <DownloadButton manifest={manifest} label="linux" primary>
            Linux (x86-64)
          </DownloadButton>
          <DownloadButton manifest={manifest} label="linux-arm64">
            Linux (ARM64)
          </DownloadButton>
        </div>
        <p className="muted small">
          Unzip it and run <code>./photo-select</code> in the extracted folder.
        </p>
      </details>
    </>
  )
}

function ChooseFolders({ os, setOS }) {
  return (
    <section id="folders" className="section">
      <h2>
        <span className="step-badge">2</span> Choose your photo folder
      </h2>
      <p className="section-lead">
        When the app starts, your browser opens its setup page. If it didn't, click this button on the same computer:
      </p>
      <p>
        <a className="button primary" href={LOCAL_URL} target="_blank" rel="noreferrer">
          Open Photo Select ↗
        </a>{' '}
        <span className="muted small">
          (opens <code>{LOCAL_URL}</code>, which only works while the app is running)
        </span>
      </p>

      <div className="setup-mock">
        <div className="field">
          <div className="field-label">1. Folder with all your photos</div>
          <div className="fake-input">{examplePath(os, 'source')}</div>
          <p>
            The folder (or drive) that holds your photos. Sub-folders are included, and photos are shown oldest first.
            Your originals are never changed.
          </p>
        </div>
        <div className="field">
          <div className="field-label">2. Where to put the selected photos</div>
          <div className="fake-input">{examplePath(os, 'output')}</div>
          <p>
            A new folder is suggested for you. Keepers are copied into a <b>Selected</b> folder inside it, and your
            progress is saved there too, so you can stop any time and carry on later.
          </p>
        </div>
        <div className="field">
          <div className="field-label">3. How many photos do you want to end up with?</div>
          <div className="fake-input short">500</div>
          <p>Just a goal for the progress bar. You can keep more or fewer.</p>
        </div>
        <div className="field">
          <div className="field-label">4 & 5. What happens when you swipe</div>
          <p>
            Leave these on the recommended settings: keepers are <b>copied</b>, skipped photos stay where they are.
          </p>
        </div>
        <div className="fake-button">Start swiping</div>
      </div>

      <h3>How do I find a folder's path?</h3>
      <Tabs tabs={PLATFORMS} value={os} onChange={setOS} />
      <div className="panel">
        {os === 'mac' && (
          <Steps>
            <li>
              Click <b>Choose…</b> next to the box and pick the folder. That's it.
            </li>
            <li>
              Or in Finder, right-click the folder, hold <Kbd>⌥ Option</Kbd> and click <b>Copy “…” as Pathname</b>,
              then paste it into the box.
            </li>
            <li>
              Photos on an external drive look like <code>/Volumes/DriveName/Photos</code>.
            </li>
          </Steps>
        )}
        {os === 'windows' && (
          <Steps>
            <li>Open File Explorer and find the folder with your photos.</li>
            <li>
              <b>Right-click</b> the folder → <b>Copy as path</b>. (On Windows 10, hold <Kbd>Shift</Kbd> while
              right-clicking.)
            </li>
            <li>
              Paste it into the box with <Kbd>Ctrl</Kbd> + <Kbd>V</Kbd>. Quotes around it are fine. It looks like{' '}
              <code>D:\Photos</code> or <code>C:\Users\you\Pictures</code>.
            </li>
          </Steps>
        )}
        {os === 'linux' && (
          <Steps>
            <li>
              Type or paste the full path, e.g. <code>/home/you/Pictures</code> or{' '}
              <code>/media/you/Drive/Photos</code>.
            </li>
            <li>
              In most file managers, <Kbd>Ctrl</Kbd> + <Kbd>L</Kbd> shows the current folder's path so you can copy
              it.
            </li>
          </Steps>
        )}
      </div>
    </section>
  )
}

function examplePath(os, kind) {
  const paths = {
    mac: ['/Volumes/MyDrive/Photos', '/Volumes/MyDrive/Photos - Selected'],
    windows: ['D:\\Photos', 'D:\\Photos - Selected'],
    linux: ['/media/me/MyDrive/Photos', '/media/me/MyDrive/Photos - Selected'],
  }
  return paths[os][kind === 'source' ? 0 : 1]
}

function Swipe() {
  return (
    <section id="swipe" className="section">
      <h2>
        <span className="step-badge">3</span> Swipe on your phone
      </h2>
      <div className="two-col">
        <div>
          <Steps>
            <li>
              After <b>Start swiping</b>, the setup page shows an address like <code>http://192.168.1.23:8787</code>.
            </li>
            <li>
              On your phone, join the <b>same Wi-Fi</b> as the computer and type that address into the browser.
            </li>
            <li>
              Swipe! Several phones can join at once and they stay in sync.
            </li>
            <li>
              <b>On the TV:</b> iPhone Control Centre → <b>Screen Mirroring</b> → your TV (or Cast on Android). Turn
              the phone sideways to fill the screen.
            </li>
            <li>
              <b>Tip:</b> on iPhone, Share → <b>Add to Home Screen</b> makes it a full-screen app.
            </li>
          </Steps>
        </div>
        <table className="controls">
          <thead>
            <tr>
              <th>Action</th>
              <th>Phone</th>
              <th>Keyboard</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="c-yes">Keep</td>
              <td>swipe right / ♥</td>
              <td>
                <Kbd>→</Kbd>
              </td>
            </tr>
            <tr>
              <td className="c-no">Skip</td>
              <td>swipe left / ✕</td>
              <td>
                <Kbd>←</Kbd>
              </td>
            </tr>
            <tr>
              <td className="c-maybe">Maybe (see again later)</td>
              <td>swipe up / ?</td>
              <td>
                <Kbd>↑</Kbd>
              </td>
            </tr>
            <tr>
              <td>Undo last</td>
              <td>↶</td>
              <td>
                <Kbd>Z</Kbd>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  )
}

function Files() {
  const items = [
    ['🔒', 'Nothing is uploaded', 'The app runs on your computer. Phones on your Wi-Fi see photos only while it is running. This website only hosts the download.'],
    ['🗂', 'Originals stay put', 'Keepers are copied into Selected/, keeping their sub-folders. Skipped photos are just remembered.'],
    ['💾', 'Stop any time', 'Progress is saved after every swipe. Start the app again, pick the same folders and carry on.'],
    ['📊', 'Get a spreadsheet', 'Download decisions (CSV) on the setup page lists every photo and what you decided.'],
  ]
  return (
    <section className="section">
      <h2>What happens to my photos?</h2>
      <div className="cards four">
        {items.map(([icon, title, text]) => (
          <div className="card" key={title}>
            <div className="card-icon">{icon}</div>
            <h3>{title}</h3>
            <p>{text}</p>
          </div>
        ))}
      </div>
      <p className="muted small">
        Works with JPEG, PNG, HEIC (iPhone), GIF, WebP, TIFF and camera RAW. Videos are skipped.
      </p>
    </section>
  )
}

function Help({ os }) {
  const uninstall = {
    mac: (
      <>
        Delete the <code>PhotoSelect</code> folder in your home folder and the <b>Photo Select</b> icon on your Desktop.
      </>
    ),
    windows: (
      <>
        Delete the <b>Photo Select</b> shortcut on your Desktop and the folder <code>%LOCALAPPDATA%\PhotoSelect</code>{' '}
        (paste that into File Explorer's address bar).
      </>
    ),
    linux: (
      <>
        Delete the <code>~/PhotoSelect</code> folder.
      </>
    ),
  }[os]

  const faqs = [
    [
      'My phone can’t open the address',
      <>
        Make sure the phone and computer are on the <b>same Wi-Fi</b> (not a guest network) and the app is still
        running. If you clicked <i>Deny</i> on the network question: on Mac go to System Settings → Network → Firewall
        → Options and allow photo-select; on Windows run the app again and allow it, or search “Allow an app through
        Windows Firewall”.
      </>,
    ],
    [
      'Nothing happened after I pasted the command',
      <>
        Make sure you pressed <Kbd>Return</Kbd>/<Kbd>Enter</Kbd>. If the browser didn't open, click{' '}
        <a href={LOCAL_URL} target="_blank" rel="noreferrer">
          Open Photo Select
        </a>{' '}
        yourself. If you see an error, copy the command again from this page and retry.
      </>,
    ],
    [
      '“Photo folder not found”',
      'The drive isn’t plugged in or was renamed, or there is a typo in the path. Plug it in and click Start swiping again.',
    ],
    [
      'The Choose… button does nothing',
      'Choose… only works on a Mac. On Windows and Linux, paste the folder path into the box (see “How do I find a folder’s path?” above).',
    ],
    [
      'Some photos say “Preview not available”',
      'On Windows and Linux, camera RAW files can’t be shown in the browser. You can still swipe them, and on a Mac they show fine.',
    ],
    [
      'How do I carry on later?',
      'Start the app again (Desktop icon), choose the same two folders and click Start swiping. It continues where you left off.',
    ],
    [
      'How do I start over?',
      'Quit the app and delete the hidden .photo-select folder inside your output folder.',
    ],
    ['How do I update?', 'Run the install command from this page again. Your progress is kept.'],
    ['How do I quit or uninstall?', <>To quit, close the Terminal/black window. To uninstall: {uninstall}</>],
  ]

  return (
    <section id="help" className="section">
      <h2>Help</h2>
      <div className="faq">
        {faqs.map(([q, a]) => (
          <details key={q}>
            <summary>{q}</summary>
            <div>{a}</div>
          </details>
        ))}
      </div>
    </section>
  )
}

function Footer({ manifest }) {
  return (
    <footer className="footer">
      <span>📸 Photo Select{manifest?.version ? ` · ${manifest.version}` : ''}</span>
      <span>Made for friends. Your photos stay on your computer.</span>
    </footer>
  )
}
