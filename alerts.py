HIGH_PRIORITY="high"
LOW_PRIORITY="low"

ACTIVE_ALERTS = [
  {
    "id":1,
    "active": True,
    "priority": HIGH_PRIORITY,
    "en": {
      "title": "Network Issues",
      "short_description": "Due to ongoing issues with the Nano network, your transactions may be delayed.",
      "long_description": "Due to ongoing issues with the Nano network, your transactions may be delayed.\n\nAnother paragraph",
      "link": "https://appditto.com/blog",
    }
  }
]

def get_active_alert(lang: str):
  for a in ACTIVE_ALERTS:
    active = a["active"]
    if active:
      if lang not in a:
        lang = 'en'
      return [{
        "id": a["id"],
        "priority": a["priority"],
        "active":a["active"],
        "title": a[lang]["title"],
        "short_description": a[lang]["short_description"],
        "long_description": a[lang]["long_description"],
        "link": a[lang]["link"]
      }]
  return []
