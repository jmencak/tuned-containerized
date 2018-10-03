#!/usr/bin/env python

import yaml
import os

tuned_profiles_cm="/var/lib/tuned/profiles-data/tuned-profiles.yaml"
tuned_profiles_dir="/etc/tuned"

with open(tuned_profiles_cm, 'r') as stream:
  try:
    d = yaml.load(stream)
    for key in d:
      profile_dir="%s/%s" % (tuned_profiles_dir, key)
      profile_file="%s/%s" % (profile_dir, "tuned.conf")
      try:
        if not os.path.exists(profile_dir):
          os.makedirs(profile_dir)
      except OSError as exc:
        raise OSError("Can't create tuned profile directory '%s': %s" % (profile_dir, exc))

      try:
        with open(profile_file, 'w') as f:
          f.write(d[key])
      except IOError as exc:
        raise IOError("Can't create tuned profile file '%s': %s" % (profile_file, exc))

  except yaml.YAMLError as exc:
    raise IOError("Can't parse tuned profiles ConfigMap file '%s': %s" % (tuned_profiles_cm, exc))
