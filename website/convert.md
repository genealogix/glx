---
title: GEDCOM and GLX converter
description: Convert genealogy files on your device between GEDCOM and single-file GLX.
sidebar: false
aside: false
---

<script setup>
import Converter from './converter/Converter.vue'
</script>

# GEDCOM and GLX converter

Convert a genealogy file on your device and download the result. Your file and any conversion warnings stay in your browser. The site downloads the converter when you start.

<Converter />

## Supported files

Use a GEDCOM 5.5.1 or 7.0 file (`.ged`) or a single-file GLX archive (`.glx`). When exporting to GEDCOM, choose the version your destination application accepts.

The converter can carry over media references, but it does not fetch or copy the media files. Use the [GLX CLI](../glx/README.md#installation) for archive folders, GEDZIP packages, or large files that strain the browser.

Some details may be lost or represented differently in the other format. Keep your original file, review any warnings, and open the result in your destination application. Use the [GLX CLI validator](../docs/cli/glx_validate.md) to check an archive.
