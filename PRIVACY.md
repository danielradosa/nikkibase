# Privacy

Last updated: 2026-09-30

NikkiBase is an unofficial Love Nikki fan project run by Daniel Radosa. This
page describes what happens to your data when you use it.

## Your wardrobe

- A wardrobe file you import is read by your own browser. Its contents are not
  uploaded to NikkiBase: the app has no upload feature, the NikkiBase server
  accepts only requests to fetch pages and data (and your notification choices,
  if you turn notifications on), and the page's security policy stops it from
  contacting any other server.
- Your wardrobe is saved in your browser's IndexedDB storage, on your own
  device, so it is still there next time. "Forget my wardrobe" deletes it, as
  does clearing this site's data in your browser.
- Items you tick by hand are saved the same way.
- "Save a copy" downloads a small text file listing the items you own. Your
  browser saves it wherever it keeps downloads; NikkiBase never sees it.
  "Copy code" puts the same list on your clipboard as text, for you to paste
  wherever you choose. A code you paste in is read by your own browser, like a
  wardrobe file.

## Using NikkiBase offline

- After your first visit, your browser keeps a copy of NikkiBase's own pages,
  scripts and item data on your device, so the site opens without a
  connection. The copy holds nothing personal and is refreshed when you visit
  online. Clearing this site's data in your browser removes it.

## Notifications (only if you turn them on)

- Notifications are off until you switch them on with the bell at the top of
  the page. Your browser then asks for your permission.
- Turning them on makes your browser create a push address for NikkiBase with
  its push service (Google for Chrome and Android, Mozilla for Firefox, Apple
  for Safari, Microsoft for Edge), plus two keys used to encrypt messages to
  it. The NikkiBase server keeps that address, the two keys, which topics you
  picked (new items, new stages, fixes) and the date you turned them on.
  Nothing else: no name, email address, account or wardrobe.
- NikkiBase sends one message to confirm they are on, and after that only when
  its data changes in a way you picked. Messages are encrypted, so the push
  service delivers them without being able to read them. The push service
  handles delivery under its own privacy policy.
- While notifications are on, your browser sends the same push address and
  choices again about once a day when you open NikkiBase, so the server's copy
  stays current.
- Turning every switch off deletes what the server kept. If the push service
  says the address no longer works (for example after you clear this site's
  data), the server deletes it the next time it sends. Blocking notifications
  for this site in your browser's settings stops them at once.

## What NikkiBase does not do

- No accounts, no cookies, no analytics, no advertising and no tracking.
- Nothing is loaded from other sites: the fonts, scripts and data are all
  served by NikkiBase itself.

## What the hosting provider sees

- NikkiBase is hosted on Railway. Like any website, the server and its host
  receive ordinary request information — your IP address, browser type, the
  address requested and the time — and the host may keep this in logs to run
  and secure the service. The NikkiBase server itself adds no logging of
  visitors. Railway's own policy: https://railway.com/legal/privacy

## Links to other sites

- The credits and footer link to other sites, such as the Love Nikki Wiki,
  GitHub and Nikki Calc. They have their own privacy policies.

## Questions

- Open an issue at https://github.com/danielradosa/nikkibase/issues.
- Or email nikkibaseproject@gmail.com.
