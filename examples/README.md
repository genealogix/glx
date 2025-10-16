# GENEALOGIX Examples

This directory contains complete, working GENEALOGIX archives
demonstrating various features and use cases.

## Quick Links

- [Minimal](minimal/) - Smallest valid archive
- [Basic Family](basic-family/) - Simple nuclear family
- [Complex Relationships](complex-relationships/) - Chosen family, adoption
- [Evidence-Based](evidence-based/) - Multiple assertions with sources
- [Oral History](oral-history/) - Oral tradition documentation
- [Large Scale](large-scale/) - Performance testing (10,000+ persons)

## Running Examples

Each example includes a README explaining its purpose and a
`test.sh` script to validate it works correctly.

```bash
cd examples/minimal
./test.sh
```

## Contributing Examples

Examples should be:
- Complete: Include all necessary files
- Valid: Pass `glx validate`
- Documented: README explaining what it demonstrates
- Tested: Include test script verifying correctness


