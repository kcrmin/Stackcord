import hashlib
import pathlib
import tempfile
import unittest

import verify_staged_release


class StagedReleaseTest(unittest.TestCase):
    def test_requires_all_archives_signature_sbom_and_manifests(self):
        with tempfile.TemporaryDirectory() as directory:
            dist = pathlib.Path(directory)
            lines = []
            for target in verify_staged_release.TARGETS:
                path = dist / f"orchestrator_1.0.0_{target}"
                path.write_bytes(target.encode())
                lines.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}")
            (dist / "checksums.txt").write_text("\n".join(lines) + "\n")
            self.assertIn("Sigstore checksum bundle is absent", verify_staged_release.verify(dist))
            (dist / "checksums.txt.sigstore.json").write_text("{}")
            (dist / "orchestrator_1.0.0.sbom.spdx").write_text("SPDXVersion: SPDX-2.3")
            (dist / "package-manifests" / "homebrew").mkdir(parents=True)
            (dist / "package-manifests" / "winget").mkdir(parents=True)
            (dist / "package-manifests" / "homebrew" / "orchestrator.rb").write_text("class Orchestrator; end")
            (dist / "package-manifests" / "winget" / "manifest.yaml").write_text("ManifestType: version")
            self.assertEqual([], verify_staged_release.verify(dist))


if __name__ == "__main__":
    unittest.main()
