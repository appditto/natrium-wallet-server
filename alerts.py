from datetime import datetime, timezone

HIGH_PRIORITY="high"
LOW_PRIORITY="low"

ACTIVE_ALERTS = [
  {
    "id":1,
    "active": True,
    "priority": HIGH_PRIORITY,
                             # yyyy, M,  D,  H,  M,  S, MS
    "timestamp": int((datetime(2020, 3, 24, 1, 0, 0, 0, tzinfo=timezone.utc) - datetime(1970, 1, 1, tzinfo=timezone.utc)).total_seconds() * 1000),
    "en": {
      "title": "Network Issues",
      "short_description": "Due to ongoing issues with the Nano network, your transactions may be delayed.",
      "long_description": "The Nano network is experiencing issues caused by a prolonged, ongoing period of spam transactions.\n\nTransactions may be significantly delayed, up to several days. We will keep our users updated with new information as the Nano team provides it.\n\nYou can read more by tapping \"Read more\" below.\n\nNatrium is not associated with Nano Foundation or its developers, so even though our service is working normally it still depends on the Nano network to behave normally. We appreciate your patience during this time.",
      "link": "https://appditto.com/blog/natrium-status-and-state-of-the-nano-network",
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
        "link": a[lang]["link"],
        "timestamp": a["timestamp"]
      }]
  return []
