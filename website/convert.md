---
title: GEDCOM and GLX converter
description: Convert genealogy files on your device between GEDCOM and GLX.
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

When converting a GEDCOM 5.5.1 or 7.0 file (`.ged`), choose **Single file (.glx)** or **Multi-file archive (.zip)**. Extract the ZIP to a folder to open it as a GLX archive.

To export to GEDCOM, select a single-file GLX archive (`.glx`) and choose the GEDCOM version your destination application accepts.

Downloads can include media references. Keep the original media files with your archive. Use the [GLX CLI](../glx/README.md#installation) to export an archive folder, import GEDZIP packages, or handle files that strain the browser.

Some details may be lost or represented differently in the other format. Keep your original file, review any warnings, and open the result in your destination application. Use the [GLX CLI validator](../docs/cli/glx_validate.md) to check an archive.
