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
import type { PricingModel } from '@/features/pricing/types'

export const SEEDANCE_MODELS = ['doubao-seedance-2.5', 'doubao-seedance-2.0']

// Only the installed CCG provider uses this JSON request contract.
export function isCcgSeedanceModel(model: PricingModel): boolean {
  return (
    SEEDANCE_MODELS.includes(model.model_name) &&
    model.billing_plugin_variants?.length === 1 &&
    model.billing_plugin_variants[0].plugin_key === 'ccg-seedance'
  )
}

export const API_EXAMPLES = {
  curl: `curl "https://gmvupup.top/v1/videos" \\
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\
  -H "Content-Type: application/json" \\
  --max-time 240 \\
  -d '{
    "model": "doubao-seedance-2.5",
    "userPrompt": "A sneaker floating above lilac dunes, slow camera movement.",
    "duration": 5,
    "quality": "720p",
    "aspectRatio": "16:9"
  }'`,
  python: `import os
import requests

response = requests.post(
    "https://gmvupup.top/v1/videos",
    headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},
    json={
        "model": "doubao-seedance-2.5",
        "userPrompt": "A sneaker floating above lilac dunes, slow camera movement.",
        "duration": 5,
        "quality": "720p",
        "aspectRatio": "16:9"
    },
    timeout=240,
)
response.raise_for_status()
task = response.json()
print(task)  # Save task["id"] before polling.`,
  javascript: `const response = await fetch("https://gmvupup.top/v1/videos", {
  method: "POST",
  headers: {
    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    "model": "doubao-seedance-2.5",
    "userPrompt": "A sneaker floating above lilac dunes, slow camera movement.",
    "duration": 5,
    "quality": "720p",
    "aspectRatio": "16:9"
  }),
  signal: AbortSignal.timeout(240000),
});
if (!response.ok) throw new Error("HTTP " + response.status);
const task = await response.json();
console.log(task); // Save task.id before polling.`,
}
