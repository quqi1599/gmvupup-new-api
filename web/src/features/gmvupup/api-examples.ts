/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
// GMVUPUP brand customization, 2026-11-08.
// These examples document the local ccg-seedance adapter, not a live service claim.
export const API_EXAMPLES = {
  curl: `curl https://gmvupup.top/v1/videos \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "doubao-seedance-2.5",
    "prompt": "A sneaker floating above lilac dunes",
    "seconds": 5,
    "size": "1280x720"
  }'`,
  python: `import os
import requests

response = requests.post(
    "https://gmvupup.top/v1/videos",
    headers={"Authorization": f"Bearer {os.environ['GMVUPUP_API_KEY']}"},
    json={
        "model": "doubao-seedance-2.5",
        "prompt": "A sneaker floating above lilac dunes",
        "seconds": 5, "size": "1280x720",
    },
    timeout=60,
)
response.raise_for_status()
task = response.json()  # Save the task ID, then poll its status.`,
  javascript: `const response = await fetch("https://gmvupup.top/v1/videos", {
  method: "POST",
  headers: {
    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "doubao-seedance-2.5",
    prompt: "A sneaker floating above lilac dunes",
    seconds: 5, size: "1280x720",
  }),
});
if (!response.ok) throw new Error("HTTP " + response.status);
const task = await response.json(); // Save the ID before polling.`,
}
