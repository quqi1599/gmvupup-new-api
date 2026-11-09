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
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js'
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js'
import { OutputPass } from 'three/addons/postprocessing/OutputPass.js'
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js'
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js'

import './style.css'

if (new URLSearchParams(location.search).get('embedded') === '1') {
  document.body.classList.add('embedded')
}

const platforms = [
  {
    name: 'TikTok Shop',
    short: 'TikTok',
    mark: '♪',
    color: '#65e8dd',
    region: '内容连接世界',
    type: 'CONTENT COMMERCE',
    desc: '从一条短视频，到一个全球爆款。让内容成为生意的起点。',
    category: '短视频 · 直播 · 兴趣消费',
    ticket: 129,
    angle: 2.7,
    height: 2.3,
  },
  {
    name: 'Temu',
    short: 'Temu',
    mark: 'T',
    color: '#ff9a49',
    region: '好物跨越山海',
    type: 'VALUE COMMERCE',
    desc: '让一件好商品，越过距离与价格的边界，抵达更多人的日常。',
    category: '全品类 · 供应链 · 全球零售',
    ticket: 79,
    angle: 3.59,
    height: 1.6,
  },
  {
    name: 'SHEIN',
    short: 'SHEIN',
    mark: 'S',
    color: '#ebd9ff',
    region: '时尚没有边界',
    type: 'FASHION COMMERCE',
    desc: '把灵感变成设计，把设计带向世界。每一次上新，都是新的可能。',
    category: '时尚 · 服饰 · 柔性供应',
    ticket: 169,
    angle: 4.48,
    height: 3.1,
  },
  {
    name: 'Amazon',
    short: 'Amazon',
    mark: 'a',
    color: '#f3c564',
    region: '从中国，到全球',
    type: 'GLOBAL MARKETPLACE',
    desc: '从第一件包裹，到一个被世界记住的品牌。长期生意，始于每次交付。',
    category: '品牌 · 精品 · 全球履约',
    ticket: 299,
    angle: 5.38,
    height: 1.9,
  },
  {
    name: 'Ozon',
    short: 'Ozon',
    mark: 'O',
    color: '#689fff',
    region: '向北，打开新市场',
    type: 'EURASIAN COMMERCE',
    desc: '跨越欧亚大陆，连接新的需求与日常。让好商品找到新的远方。',
    category: '欧亚 · 跨境 · 本地需求',
    ticket: 219,
    angle: 0,
    height: 2.7,
  },
  {
    name: 'Shopee',
    short: '虾皮',
    mark: 'S',
    color: '#ff7a64',
    region: '热爱，生生不息',
    type: 'SOUTHEAST ASIA',
    desc: '从热带海岛到繁忙都市，连接东南亚的生活，也连接每个卖家的机会。',
    category: '东南亚 · 移动购物 · 日用消费',
    ticket: 89,
    angle: 0.9,
    height: 1.5,
  },
  {
    name: 'Mercado Libre',
    short: '美客多',
    mark: 'M',
    color: '#f2de67',
    region: '跨越半球的生意',
    type: 'LATIN AMERICAN COMMERCE',
    desc: '越过太平洋，让中国卖家的创造力，遇见拉美市场的生命力。',
    category: '拉美 · 全品类 · 跨境品牌',
    ticket: 259,
    angle: 1.8,
    height: 2.4,
  },
]
const icons = {
  globe:
    '<circle cx="12" cy="12" r="9"/><ellipse cx="12" cy="12" rx="4" ry="9"/><path d="M3 12h18M5 7h14M5 17h14"/>',
  volume:
    '<path d="m11 5-6 4H2v6h3l6 4zM15 8c3 2 3 6 0 8M18 5c5 4 5 10 0 14"/>',
  mute: '<path d="m11 5-6 4H2v6h3l6 4zM16 9l6 6M22 9l-6 6"/>',
  help: '<circle cx="12" cy="12" r="9"/><path d="M9 8a3 3 0 0 1 6 1c0 2-3 2-3 4M12 17h.01"/>',
  screen: '<path d="M8 3H3v5M16 3h5v5M21 16v5h-5M8 21H3v-5"/>',
  camera: '<path d="M8 6 9 3h6l1 3h5v14H3V6z"/><circle cx="12" cy="12" r="4"/>',
  mouse: '<rect x="6" y="2" width="12" height="20" rx="6"/><path d="M12 2v6"/>',
  orbit:
    '<ellipse cx="12" cy="12" rx="10" ry="4" transform="rotate(-30 12 12)"/><circle cx="12" cy="12" r="3"/>',
  reset: '<path d="M3 10a9 9 0 1 1 2 8M3 4v6h6"/>',
  pause: '<path d="M8 5v14M16 5v14"/>',
  play: '<path d="m7 4 14 8-14 8z"/>',
  launch: '<path d="m3 11 18-8-8 18-2-8zM11 13l5-5"/>',
  close: '<path d="m6 6 12 12M18 6 6 18"/>',
}
const svg = (name) =>
  `<svg viewBox="0 0 24 24" aria-hidden="true">${icons[name]}</svg>`

// Original, locally drawn wordmarks: the whole experience runs without an asset CDN.
function brandCanvas(p, index) {
  const c = document.createElement('canvas')
  c.width = c.height = 256
  const x = c.getContext('2d')
  x.textAlign = 'center'
  x.textBaseline = 'middle'
  x.fillStyle = p.color
  if (index === 0) {
    x.font = 'bold 205px Arial'
    x.fillStyle = '#ff527c'
    x.fillText('♪', 136, 134)
    x.fillStyle = '#65e8dd'
    x.fillText('♪', 123, 125)
    x.fillStyle = '#fff'
    x.fillText('♪', 130, 129)
  } else if (index === 1) {
    x.font = '900 68px Arial'
    x.fillText('TEMU', 128, 134)
    x.lineWidth = 5
    x.strokeStyle = p.color
    x.beginPath()
    x.arc(128, 78, 35, Math.PI, Math.PI * 2)
    x.stroke()
  } else if (index === 2) {
    x.font = '600 62px Arial'
    x.fillText('SHEIN', 128, 132)
  } else if (index === 3) {
    x.font = 'bold 185px Georgia'
    x.fillText('a', 124, 114)
    x.lineWidth = 10
    x.strokeStyle = p.color
    x.beginPath()
    x.moveTo(62, 186)
    x.quadraticCurveTo(134, 219, 198, 176)
    x.stroke()
    x.beginPath()
    x.moveTo(177, 176)
    x.lineTo(201, 175)
    x.lineTo(196, 200)
    x.stroke()
  } else if (index === 4) {
    x.font = '900 68px Arial'
    x.fillText('OZON', 128, 132)
  } else if (index === 5) {
    x.lineWidth = 8
    x.strokeStyle = p.color
    x.strokeRect(57, 81, 142, 142)
    x.beginPath()
    x.arc(128, 83, 37, Math.PI, 0)
    x.stroke()
    x.font = '110px Arial'
    x.fillText('S', 129, 158)
  } else {
    x.lineWidth = 7
    x.strokeStyle = p.color
    x.beginPath()
    x.ellipse(128, 111, 94, 61, 0, 0, Math.PI * 2)
    x.stroke()
    x.beginPath()
    x.moveTo(50, 105)
    x.lineTo(94, 93)
    x.lineTo(119, 105)
    x.lineTo(138, 94)
    x.lineTo(183, 121)
    x.lineTo(160, 146)
    x.lineTo(115, 124)
    x.lineTo(97, 140)
    x.lineTo(65, 121)
    x.stroke()
    x.font = 'bold 29px Arial'
    x.fillText('mercado libre', 128, 205)
  }
  return c
}
platforms.forEach((p, i) => {
  p.logo = brandCanvas(p, i)
  p.image = p.logo.toDataURL()
})
document.querySelector('#app').innerHTML = `
<canvas id="world" tabindex="0" aria-label="电商 GMV 3D 交互场景，拖动旋转，滚轮缩放，点击平台选择，空格发射订单"></canvas>
<div class="vignette"></div><div class="grain"></div>
<header class="topbar"><div class="brand"><span class="brand-icon">${svg('globe')}</span>电商 GMV<small>THE COMMERCE UNIVERSE</small></div><span class="edition">INTERACTIVE ATLAS / VOL. 001</span><div class="header-actions"><span class="live"><i></i>实时模拟</span><button id="audio" class="icon-btn" title="开启声音" aria-label="开启声音" aria-pressed="false">${svg('mute')}</button><button id="capture" class="icon-btn" title="保存画面" aria-label="保存画面">${svg('camera')}</button><button id="cinema" class="icon-btn" title="沉浸模式" aria-label="沉浸模式">${svg('screen')}</button><button id="help" class="icon-btn" title="玩法与操作" aria-label="玩法与操作">${svg('help')}</button></div></header>
<section class="intro"><div class="eyebrow">中国卖家 · 全球舞台</div><h1>G<em>M</em>V<span style="font:12px ui-monospace,monospace;letter-spacing:0;color:#947b58;margin-left:10px">/ 01</span></h1><h2>生意，无远弗届。</h2><p>让每一份创造，被世界看见。<br>让每一笔订单，点亮新的远方。</p><span class="coordinate">ORIGIN 31° N / 121° E → EVERYWHERE</span></section>
<span class="side-line">FROM ONE IDEA TO EVERY CORNER OF THE WORLD</span>
<aside class="inspector" aria-label="平台详情"><div class="section-top"><span>DESTINATION</span><span id="platform-index">01 / 07</span></div><div class="platform-name"><div class="platform-mark" id="platform-mark">♪</div><div><h3 id="platform-name">TikTok Shop</h3><small id="platform-region">内容连接世界</small></div></div><p class="platform-desc" id="platform-desc"></p><dl><div><dt>商业形态</dt><dd id="platform-type"></dd></div><div><dt>模拟订单</dt><dd id="platform-orders">0 笔</dd></div><div><dt>模拟客单价</dt><dd id="platform-ticket"></dd></div></dl><div class="separator"></div><div class="route-status"><span>ORDER CHANNEL</span><strong>CONNECTED ↗</strong></div><div class="meter"><span id="route-meter"></span></div></aside>
<div class="labels" aria-hidden="false">${platforms.map((p, i) => `<div class="node-label" data-node="${i}" style="--node:${p.color}"><button data-select="${i}" aria-label="聚焦 ${p.name}"><i></i>${p.name}</button><small>${p.type}</small></div>`).join('')}</div>
<div class="core-label"><span>THE GMV ENGINE</span><small>创造 · 连接 · 增长</small></div>
<section class="readout" aria-label="模拟交易数据"><span>GROSS MERCHANDISE VALUE / 模拟成交额</span><div class="amount"><small>¥</small><span id="gmv">0</span></div><div class="substats"><span><b id="orders">0</b> ORDERS</span><span><b id="combo">×1</b> PULSE</span></div><svg class="spark" viewBox="0 0 160 25"><defs><linearGradient id="sg" x2="0" y2="1"><stop stop-color="#edc58d" stop-opacity=".2"/><stop offset="1" stop-color="#edc58d" stop-opacity="0"/></linearGradient></defs><path class="fill" d="M0 24L0 23L20 22L32 16L44 20L58 14L70 16L85 9L100 12L115 5L130 8L145 2L160 4L160 25Z"/><path d="M0 23L20 22L32 16L44 20L58 14L70 16L85 9L100 12L115 5L130 8L145 2L160 4"/></svg></section>
<section class="goal" aria-label="探索进度"><div class="goal-head"><span id="goal-title">点亮七个平台，连接世界</span><b id="goal-count">0 / 7</b></div><div class="goal-dots">${platforms.map(() => '<i></i>').join('')}</div><p id="goal-copy">选择平台，发射订单。<br>连续发射，触发增长共振。</p></section>
<footer class="bottom"><div class="dock-header"><span>01 — 07 / GLOBAL COMMERCE NETWORK</span><span>一个起点，七种可能。选择你的下一站 ↙</span></div><nav class="dock" aria-label="平台选择">${platforms.map((p, i) => `<button class="platform ${i === 0 ? 'active' : ''}" data-select="${i}" style="--node:${p.color}" aria-pressed="${i === 0}"><img src="${p.image}" alt=""/><span>${p.short}<small>${String(i + 1).padStart(2, '0')} / ${['CONTENT', 'VALUE', 'FASHION', 'GLOBAL', 'EURASIA', 'SEA', 'LATAM'][i]}</small></span><b class="done" id="done-${i}"></b></button>`).join('')}</nav><div class="toolbar"><div class="interaction-hint"><span>${svg('mouse')}拖动探索</span><span>滚轮缩放</span><span><kbd class="key">R</kbd> 返回全景</span><span><kbd class="key">H</kbd> 隐藏界面</span></div><div class="tools"><button class="text-btn active" id="orbit-toggle" aria-pressed="true">${svg('orbit')}巡航</button><button class="text-btn" id="reset" title="返回全景（R）">${svg('reset')}全景</button><button class="text-btn" id="pause" aria-label="暂停模拟">${svg('pause')}暂停</button><button class="text-btn" id="speed" aria-label="模拟速度 1 倍">1×</button><button class="launch" id="launch">${svg('launch')}发射订单<kbd class="key">SPACE</kbd></button></div></div></footer>
<button class="exit-cinema" id="exit-cinema">退出沉浸模式 · H / ESC</button>
<div class="toast" role="status" aria-live="polite"></div>
<div class="modal" id="help-modal" role="dialog" aria-modal="true" aria-labelledby="help-title" hidden><div class="modal-content"><div class="modal-top"><span class="eyebrow">HOW TO EXPLORE</span><button class="icon-btn" id="close-help" aria-label="关闭">${svg('close')}</button></div><h2 id="help-title">下一笔生意，由你点亮。</h2><p>这是一个可玩的全球电商宇宙。选择七个平台，向它们发射订单；连续发射积累连击，点亮全部平台后，触发全球共振。</p><div class="help-list"><div><span>旋转 / 缩放</span><span>鼠标拖动 / 滚轮 · 双指触控</span></div><div><span>选择平台</span><span>点击立方体、标签或底部平台</span></div><div><span>发射订单 / 返回全景</span><span><kbd class="key">SPACE</kbd> / <kbd class="key">R</kbd></span></div><div><span>沉浸模式 / 退出</span><span><kbd class="key">H</kbd> / <kbd class="key">ESC</kbd></span></div><div><span>模拟速度</span><span>点击 1× 切换至 2× / 0.5×</span></div></div><p class="fine">全部金额、订单和流量均为本地交互模拟，不代表平台经营数据。品牌名称仅用于识别；本作品与上述平台无隶属关系。声音默认关闭。</p><button id="start" class="launch" style="margin:22px 0 0;width:100%">开始探索 ${svg('launch')}</button></div></div>
<div class="loader"><div class="loader-orbit"></div><span class="loader-title">CONNECTING THE WORLD</span><p>正在点亮你的电商宇宙</p></div>`

const $ = (s) => document.querySelector(s)
const nodeLabels = [...document.querySelectorAll('.node-label')]
let labelHalves = []
const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
const mobile = () => innerWidth < 700
let selected = 0,
  paused = false,
  speed = 1,
  autoOrbit = !reduced,
  cinema = false,
  audioOn = false,
  audioContext
let total = 0,
  orderCount = 0,
  combo = 1,
  lastLaunch = -10,
  lastFrame = 0,
  elapsed = 0,
  renderTime = 0,
  autoTime = 0
let focusTween = null,
  toastTimer,
  activePointers = new Map(),
  completionTime = -100
const activated = new Set(),
  platformOrders = platforms.map(() => 0)
const stats = { frames: 0, fps: 0, renderer: '', ready: false, errors: [] }
let renderer, scene, camera, controls, composer, bloom
const nodes = [],
  curves = [],
  packets = [],
  sparks = [],
  ripples = [],
  rotors = [],
  lights = [],
  hoverables = []
const raycaster = new THREE.Raycaster(),
  pointer = new THREE.Vector2(),
  tmp = new THREE.Vector3()
const gold = new THREE.Color('#edc58d')

const sphereGeo = new THREE.SphereGeometry(1, 14, 10),
  boxGeo = new THREE.BoxGeometry(1, 1, 1)
function basic(color, opacity = 1) {
  return new THREE.MeshBasicMaterial({
    color,
    transparent: opacity < 1,
    opacity,
    depthWrite: opacity >= 1,
  })
}
function standard(color, metalness = 0.7, roughness = 0.3) {
  return new THREE.MeshStandardMaterial({ color, metalness, roughness })
}
function mesh(geo, mat, parent = scene) {
  const m = new THREE.Mesh(geo, mat)
  parent.add(m)
  return m
}
function ring(radius, tube, color, parent = scene, opacity = 1) {
  const m = mesh(
    new THREE.TorusGeometry(radius, tube, 8, 180),
    basic(color, opacity),
    parent
  )
  m.rotation.x = Math.PI / 2
  return m
}
function line(points, color, opacity = 0.3, parent = scene) {
  const l = new THREE.Line(
    new THREE.BufferGeometry().setFromPoints(points),
    new THREE.LineBasicMaterial({
      color,
      transparent: true,
      opacity,
      depthWrite: false,
    })
  )
  parent.add(l)
  return l
}
function toast(text) {
  $('.toast').textContent = text
  $('.toast').classList.add('show')
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => $('.toast').classList.remove('show'), 2900)
}
function chime(freq = 520) {
  if (!audioOn) return
  try {
    audioContext ??= new AudioContext()
    audioContext.resume()
    const o = audioContext.createOscillator(),
      g = audioContext.createGain()
    o.type = 'sine'
    o.frequency.setValueAtTime(freq, audioContext.currentTime)
    o.frequency.exponentialRampToValueAtTime(
      freq * 1.5,
      audioContext.currentTime + 0.24
    )
    g.gain.setValueAtTime(0.035, audioContext.currentTime)
    g.gain.exponentialRampToValueAtTime(0.0001, audioContext.currentTime + 0.7)
    o.connect(g).connect(audioContext.destination)
    o.start()
    o.stop(audioContext.currentTime + 0.75)
  } catch {}
}
function fatal(error) {
  stats.errors.push(String(error))
  $('.loader')?.remove()
  const el = document.createElement('section')
  el.className = 'fatal'
  el.innerHTML =
    '<h1 style="font-size:60px;letter-spacing:-2px">GMV</h1><h2>暂时无法启动 3D 场景</h2><p>请使用支持 WebGL 2 的现代浏览器，并开启硬件加速。关闭占用显卡的其他标签页后可重新尝试。</p><button onclick="location.reload()">重新连接</button>'
  document.body.append(el)
  console.error(error)
}

function init() {
  const canvas = $('#world')
  renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: false,
    alpha: false,
    powerPreference: 'high-performance',
    preserveDrawingBuffer: true,
  })
  renderer.setPixelRatio(Math.min(devicePixelRatio, mobile() ? 1.5 : 1.7))
  renderer.setSize(innerWidth, innerHeight)
  renderer.toneMapping = THREE.ACESFilmicToneMapping
  renderer.toneMappingExposure = 1.05
  renderer.info.autoReset = false
  scene = new THREE.Scene()
  scene.background = new THREE.Color('#080c10')
  scene.fog = new THREE.FogExp2('#080e12', 0.01)
  camera = new THREE.PerspectiveCamera(43, innerWidth / innerHeight, 0.1, 220)
  camera.position.copy(homeCamera())
  controls = new OrbitControls(camera, canvas)
  controls.target.set(0, 1, 0)
  controls.enableDamping = true
  controls.dampingFactor = 0.055
  controls.enablePan = false
  controls.minDistance = 12
  controls.maxDistance = 105
  controls.minPolarAngle = 0.15
  controls.maxPolarAngle = Math.PI * 0.49
  controls.autoRotate = autoOrbit
  controls.autoRotateSpeed = 0.25
  controls.addEventListener('start', () => {
    focusTween = null
    setOrbit(false)
  })
  scene.add(new THREE.AmbientLight('#789baa', 0.55))
  const sun = new THREE.DirectionalLight('#ffe2af', 2.2)
  sun.position.set(3, 15, 8)
  scene.add(sun)
  const rim = new THREE.DirectionalLight('#6b9cc5', 1.4)
  rim.position.set(-15, 5, -12)
  scene.add(rim)
  const envScene = new RoomEnvironment()
  const pmrem = new THREE.PMREMGenerator(renderer)
  const environment = pmrem.fromScene(envScene, 0.04)
  scene.environment = environment.texture
  envScene.dispose()
  pmrem.dispose()
  const coreLight = new THREE.PointLight('#ffbe6f', 70, 25, 2)
  coreLight.position.set(0, 6, 0)
  scene.add(coreLight)
  buildFloor()
  buildStars()
  buildCore()
  platforms.forEach(buildPlatform)
  buildDust()
  composer = new EffectComposer(renderer)
  composer.addPass(new RenderPass(scene, camera))
  bloom = new UnrealBloomPass(
    new THREE.Vector2(innerWidth, innerHeight),
    0.65,
    0.6,
    0.68
  )
  composer.addPass(bloom)
  composer.addPass(new OutputPass())
  // Half-resolution bloom keeps the full-resolution scene crisp and makes the orbital field affordable on mobile.
  bloom.resolution.set(innerWidth * 0.65, innerHeight * 0.65)
  setupUI()
  selectPlatform(0, false)
  setOrbit(autoOrbit)
  resize()
  const gl = renderer.getContext()
  stats.renderer = gl.getParameter(gl.RENDERER)
  stats.ready = true
  window.__GMV__ = {
    snapshot: () => ({
      ready: stats.ready,
      frames: stats.frames,
      fps: stats.fps,
      selected: platforms[selected].name,
      paused,
      speed,
      autoOrbit,
      cinema,
      gmv: total,
      orders: orderCount,
      activated: [...activated],
      packets: packets.length,
      camera: camera.position.toArray(),
      target: controls.target.toArray(),
      webgl: renderer.capabilities.isWebGL2,
      drawCalls: renderer.info.render.calls,
      triangles: renderer.info.render.triangles,
      errors: stats.errors,
    }),
    projectNode: (i) => {
      const v = nodes[i].position
        .clone()
        .add(new THREE.Vector3(0, 0.55, 0))
        .project(camera)
      return {
        x: (v.x * 0.5 + 0.5) * innerWidth,
        y: (-0.5 * v.y + 0.5) * innerHeight,
      }
    },
    version: '1.0.0',
  }
  renderer.setAnimationLoop(animate)
  setTimeout(() => {
    $('.loader').classList.add('hidden')
    setTimeout(() => $('.loader').remove(), 900)
  }, 450)
  window.addEventListener('resize', resize)
  canvas.addEventListener('webglcontextlost', (e) => {
    e.preventDefault()
    stats.ready = false
    paused = true
    toast('图形连接暂时中断，正在等待恢复…')
  })
  canvas.addEventListener('webglcontextrestored', () => location.reload())
}
function homeCamera() {
  const portrait = innerWidth / innerHeight < 0.8
  return new THREE.Vector3(
    portrait ? 43 : 23,
    portrait ? 49 : 25,
    portrait ? 58 : 31
  ).multiplyScalar(innerWidth / innerHeight < 1.1 && !portrait ? 1.15 : 1)
}
function buildFloor() {
  const floor = mesh(new THREE.PlaneGeometry(500, 500), basic('#080d10'))
  floor.rotation.x = -Math.PI / 2
  floor.position.y = -2.1
  for (const r of [3.9, 6.4, 10.7, 13.3, 16.5, 21, 28]) {
    const a = ring(r, 0.009, '#4f8671', scene, r === 13.3 ? 0.28 : 0.13)
    a.position.y = -2.065
  }
  for (let i = 0; i < 100; i++) {
    const a = (i / 100) * Math.PI * 2,
      r = i % 5 === 0 ? 20.8 : 17.2
    line(
      [
        new THREE.Vector3(Math.cos(a) * 16.7, -2.05, Math.sin(a) * 16.7),
        new THREE.Vector3(Math.cos(a) * r, -2.05, Math.sin(a) * r),
      ],
      '#70907a',
      i % 5 === 0 ? 0.28 : 0.16
    )
  }
  for (let i = 0; i < 12; i++) {
    const a = (i * Math.PI) / 6
    line(
      [
        new THREE.Vector3(Math.cos(a) * 4, -2.07, Math.sin(a) * 4),
        new THREE.Vector3(Math.cos(a) * 34, -2.07, Math.sin(a) * 34),
      ],
      '#447362',
      0.12
    )
  }
  const textureCanvas = document.createElement('canvas')
  textureCanvas.width = textureCanvas.height = 256
  const x = textureCanvas.getContext('2d'),
    g = x.createRadialGradient(128, 128, 0, 128, 128, 128)
  g.addColorStop(0, 'rgba(217,162,73,.25)')
  g.addColorStop(0.35, 'rgba(182,133,60,.1)')
  g.addColorStop(1, 'rgba(0,0,0,0)')
  x.fillStyle = g
  x.fillRect(0, 0, 256, 256)
  const glow = mesh(
    new THREE.PlaneGeometry(20, 20),
    new THREE.MeshBasicMaterial({
      map: new THREE.CanvasTexture(textureCanvas),
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    })
  )
  glow.rotation.x = -Math.PI / 2
  glow.position.y = -2.02
}
function buildStars() {
  const a = [],
    colors = []
  for (let i = 0; i < 1800; i++) {
    const r = 35 + Math.random() * 75,
      angle = Math.random() * Math.PI * 2,
      z = Math.random() * 1.5 - 0.45
    a.push(Math.cos(angle) * r, z * r, Math.sin(angle) * r)
    const c = new THREE.Color(i % 5 === 0 ? '#eed2a2' : '#71959b')
    colors.push(c.r, c.g, c.b)
  }
  const g = new THREE.BufferGeometry()
  g.setAttribute('position', new THREE.Float32BufferAttribute(a, 3))
  g.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3))
  scene.add(
    new THREE.Points(
      g,
      new THREE.PointsMaterial({
        size: 0.075,
        vertexColors: true,
        transparent: true,
        opacity: 0.65,
        sizeAttenuation: true,
        depthWrite: false,
      })
    )
  )
}
const core = new THREE.Group(),
  earth = new THREE.Group()
function globePoint(lon, lat, r) {
  const a = (lon * Math.PI) / 180,
    b = (lat * Math.PI) / 180
  return new THREE.Vector3(
    r * Math.cos(b) * Math.cos(a),
    r * Math.sin(b),
    -r * Math.cos(b) * Math.sin(a)
  )
}
function pointInPolygon(x, y, poly) {
  let inside = false
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [xi, yi] = poly[i],
      [xj, yj] = poly[j]
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) {
      inside = !inside
    }
  }
  return inside
}
function buildCore() {
  scene.add(core)
  core.position.y = 2
  core.scale.setScalar(1.2)
  core.add(earth)
  earth.rotation.z = 0.19
  earth.rotation.y = -1.5
  mesh(
    new THREE.SphereGeometry(3.1, 64, 40),
    new THREE.MeshPhysicalMaterial({
      color: '#192d25',
      metalness: 0.72,
      roughness: 0.48,
      clearcoat: 0.3,
      clearcoatRoughness: 0.4,
    }),
    earth
  )
  const gridMat = new THREE.LineBasicMaterial({
    color: '#b5a270',
    transparent: true,
    opacity: 0.2,
  })
  for (let lat = -75; lat <= 75; lat += 15) {
    const pts = []
    for (let lon = 0; lon <= 360; lon += 3) pts.push(globePoint(lon, lat, 3.12))
    earth.add(
      new THREE.Line(new THREE.BufferGeometry().setFromPoints(pts), gridMat)
    )
  }
  for (let lon = 0; lon < 360; lon += 15) {
    const pts = []
    for (let lat = -90; lat <= 90; lat += 3) {
      pts.push(globePoint(lon, lat, 3.12))
    }
    earth.add(
      new THREE.Line(new THREE.BufferGeometry().setFromPoints(pts), gridMat)
    )
  }
  const land = [
    [
      [0, 35],
      [-17, 15],
      [-5, 5],
      [9, -4],
      [18, -34],
      [31, -26],
      [43, -11],
      [50, 10],
      [39, 20],
      [33, 30],
    ],
    [
      [0, 38],
      [-9, 43],
      [-6, 57],
      [15, 71],
      [37, 67],
      [52, 55],
      [50, 41],
      [26, 36],
      [13, 43],
    ],
    [
      [33, 39],
      [48, 53],
      [56, 69],
      [100, 77],
      [161, 67],
      [179, 53],
      [153, 43],
      [139, 34],
      [129, 37],
      [124, 21],
      [107, 4],
      [96, 17],
      [77, 7],
      [67, 25],
      [45, 31],
    ],
    [
      [112, -11],
      [134, -10],
      [154, -25],
      [143, -39],
      [115, -34],
    ],
    [
      [-168, 71],
      [-130, 73],
      [-108, 82],
      [-67, 61],
      [-55, 46],
      [-81, 24],
      [-87, 15],
      [-104, 22],
      [-122, 38],
      [-141, 60],
    ],
    [
      [-81, 12],
      [-60, 9],
      [-35, -5],
      [-45, -26],
      [-68, -55],
      [-78, -22],
    ],
    [
      [-55, 59],
      [-21, 65],
      [-32, 83],
      [-58, 81],
    ],
    [
      [130, 32],
      [138, 45],
      [145, 44],
      [140, 31],
    ],
    [
      [47, -13],
      [50, -15],
      [49, -26],
      [44, -24],
    ],
    [
      [95, 5],
      [105, -7],
      [123, -9],
      [116, -2],
    ],
  ]
  const positions = [],
    cols = []
  for (let lat = -57; lat < 80; lat += 2.4) {
    for (
      let lon = -180;
      lon < 180;
      lon += 2.4 / Math.max(0.3, Math.cos((lat * Math.PI) / 180))
    ) {
      if (!land.some((poly) => pointInPolygon(lon, lat, poly))) continue
      const p = globePoint(lon, lat, 3.145 + Math.random() * 0.025)
      positions.push(...p.toArray())
      const c = new THREE.Color().setHSL(
        0.1 + Math.random() * 0.025,
        0.58,
        0.4 + Math.random() * 0.25
      )
      cols.push(c.r, c.g, c.b)
    }
  }
  const geo = new THREE.BufferGeometry()
  geo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
  geo.setAttribute('color', new THREE.Float32BufferAttribute(cols, 3))
  earth.add(
    new THREE.Points(
      geo,
      new THREE.PointsMaterial({
        vertexColors: true,
        size: 0.05,
        transparent: true,
        opacity: 0.95,
        blending: THREE.AdditiveBlending,
      })
    )
  )
  // Fresnel atmosphere: transparent at the center, a thin luminous rim at the horizon.
  mesh(
    new THREE.SphereGeometry(3.22, 64, 40),
    new THREE.ShaderMaterial({
      uniforms: { tint: { value: gold } },
      vertexShader:
        'varying vec3 n; varying vec3 v; void main(){vec4 p=modelViewMatrix*vec4(position,1.0);n=normalize(normalMatrix*normal);v=normalize(-p.xyz);gl_Position=projectionMatrix*p;}',
      fragmentShader:
        'uniform vec3 tint; varying vec3 n; varying vec3 v; void main(){float rim=pow(1.0-abs(dot(normalize(n),normalize(v))),4.0);gl_FragColor=vec4(tint*1.4,rim*.42);}',
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      side: THREE.FrontSide,
    }),
    core
  )
  for (let i = 0; i < 3; i++) {
    const g = new THREE.Group()
    g.rotation.set(0.4 + i * 0.65, i * 0.7, i * 0.5)
    core.add(g)
    const r = ring(
      4.1 + i * 0.43,
      0.014,
      i === 1 ? '#8bc6b8' : '#d8ad68',
      g,
      0.65
    )
    r.rotation.x = 0
    const arc = []
    for (let k = 0; k < 90; k++) {
      const t = (k / 90) * 1.25
      arc.push(
        new THREE.Vector3(
          Math.cos(t) * (4.1 + i * 0.43),
          Math.sin(t) * (4.1 + i * 0.43),
          0
        )
      )
    }
    line(arc, '#ffe2a0', 0.9, g)
    const satellite = mesh(sphereGeo, basic('#ffe2a7'), g)
    satellite.scale.setScalar(0.065)
    satellite.position.set(4.1 + i * 0.43, 0, 0)
    rotors.push(g)
  }
  for (const [lon, lat] of [
    [121, 31],
    [114, 23],
    [116, 40],
    [-74, 40],
    [-46, -23],
    [-99, 19],
    [37, 55],
    [106, -6],
    [101, 13],
  ]) {
    const p = globePoint(lon, lat, 3.19)
    const city = mesh(sphereGeo, basic('#ffe7b6'), earth)
    city.scale.setScalar(0.055)
    city.position.copy(p)
  }
  const haloCanvas = document.createElement('canvas')
  haloCanvas.width = haloCanvas.height = 256
  const ctx = haloCanvas.getContext('2d'),
    halo = ctx.createRadialGradient(128, 128, 35, 128, 128, 128)
  halo.addColorStop(0, 'rgba(245,177,75,.21)')
  halo.addColorStop(0.45, 'rgba(232,169,76,.08)')
  halo.addColorStop(1, 'rgba(232,169,76,0)')
  ctx.fillStyle = halo
  ctx.fillRect(0, 0, 256, 256)
  const haloSprite = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: new THREE.CanvasTexture(haloCanvas),
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    })
  )
  haloSprite.scale.set(15, 15, 1)
  core.add(haloSprite)
  // The tilted annular field has individually animated specks rather than a flat opaque disk.
  const orbital = new THREE.Group()
  orbital.rotation.set(0.18, 0, 0.24)
  core.add(orbital)
  const orbitalPts = []
  for (let i = 0; i < 2400; i++) {
    const r = 4.5 + Math.pow(Math.random(), 1.8) * 2.1,
      a = Math.random() * Math.PI * 2
    orbitalPts.push(
      Math.cos(a) * r,
      (Math.random() - 0.5) * 0.045,
      Math.sin(a) * r
    )
  }
  const orbitalGeo = new THREE.BufferGeometry()
  orbitalGeo.setAttribute(
    'position',
    new THREE.Float32BufferAttribute(orbitalPts, 3)
  )
  orbital.add(
    new THREE.Points(
      orbitalGeo,
      new THREE.PointsMaterial({
        color: '#e7bf7c',
        size: 0.025,
        transparent: true,
        opacity: 0.6,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      })
    )
  )
  rotors.push(orbital)
  const base = mesh(
    new THREE.CylinderGeometry(3.7, 4, 0.28, 100),
    standard('#18231d', 0.92, 0.23)
  )
  base.position.y = -1.84
  for (let i = 0; i < 3; i++) {
    const r = ring(3.6 + i * 0.15, 0.015, '#c6a266', scene, 0.65 - i * 0.15)
    r.position.y = -1.65 + i * 0.12
  }
  const tics = new THREE.Group()
  scene.add(tics)
  for (let i = 0; i < 64; i++) {
    const a = (i / 64) * Math.PI * 2
    const bar = mesh(boxGeo, standard('#b7a47d', 0.8, 0.3), tics)
    bar.scale.set(0.028, 0.045, i % 4 === 0 ? 0.24 : 0.12)
    bar.position.set(Math.cos(a) * 4.2, -1.67, Math.sin(a) * 4.2)
    bar.rotation.y = -a + Math.PI / 2
  }
}
function buildPlatform(p, i) {
  const node = new THREE.Group()
  node.position.set(
    Math.cos(p.angle) * 11.8,
    p.height,
    Math.sin(p.angle) * 11.8
  )
  scene.add(node)
  nodes.push(node)
  const c = new THREE.Color(p.color)
  const pedestal = mesh(
    new THREE.CylinderGeometry(1.25, 1.48, 0.24, 64),
    standard('#14221f', 0.9, 0.25),
    node
  )
  pedestal.position.y = -0.7
  ring(1.26, 0.025, c, node, 0.85).position.y = -0.55
  ring(1.54, 0.009, c, node, 0.25).position.y = -0.65
  const upright = new THREE.Group()
  node.add(upright)
  upright.position.y = 0.55
  upright.rotation.y = Math.atan2(
    camera.position.x - node.position.x,
    camera.position.z - node.position.z
  )
  upright.userData.platform = true
  rotors.push(upright)
  const bodyMat = new THREE.MeshPhysicalMaterial({
    color: '#25332f',
    metalness: 0.85,
    roughness: 0.2,
    clearcoat: 1,
    emissive: c,
    emissiveIntensity: 0.045,
  })
  const body = mesh(new THREE.BoxGeometry(1.8, 1.85, 0.55), bodyMat, upright)
  hoverables.push(body)
  body.userData.index = i
  const edge = new THREE.LineSegments(
    new THREE.EdgesGeometry(body.geometry),
    new THREE.LineBasicMaterial({ color: c, transparent: true, opacity: 0.65 })
  )
  body.add(edge)
  const texture = new THREE.CanvasTexture(p.logo)
  texture.colorSpace = THREE.SRGBColorSpace
  for (const side of [-1, 1]) {
    const logo = mesh(
      new THREE.PlaneGeometry(1.5, 1.5),
      new THREE.MeshBasicMaterial({
        map: texture,
        transparent: true,
        side: THREE.DoubleSide,
        depthWrite: false,
      }),
      upright
    )
    logo.position.z = side * 0.282
    if (side < 0) logo.rotation.y = Math.PI
  }
  // A restrained beam anchors each floating trading terminal to the orbital floor.
  const beam = mesh(
    new THREE.CylinderGeometry(0.015, 0.015, p.height + 1.3, 8),
    basic(c, 0.25)
  )
  beam.position.set(node.position.x, (p.height - 2) / 2, node.position.z)
  const floorRing = ring(0.7, 0.012, c, scene, 0.32)
  floorRing.position.set(node.position.x, -2, node.position.z)
  const glow = mesh(sphereGeo, basic(c), node)
  glow.scale.setScalar(0.055)
  glow.position.y = -0.37
  const light = new THREE.PointLight(c, 8, 7, 2)
  light.position.copy(node.position).add(new THREE.Vector3(0, 2, 1))
  scene.add(light)
  lights.push(light)
  const start = new THREE.Vector3(0, 1.5, 0),
    end = node.position.clone().add(new THREE.Vector3(0, -0.4, 0))
  const curve = new THREE.CubicBezierCurve3(
    start,
    new THREE.Vector3(end.x * 0.25, 5.9, end.z * 0.25),
    new THREE.Vector3(end.x * 0.85, 5.2, end.z * 0.85),
    end
  )
  curves.push(curve)
  line(curve.getPoints(95), c, 0.26)
  const under = curve
    .getPoints(75)
    .map((v) => new THREE.Vector3(v.x, -2.02, v.z))
  line(under, c, 0.07)
  // Each route carries a small cargo procession even before the first interaction.
  for (let k = 0; k < 3; k++) spawnPacket(i, k / 3, true)
}
let dust
function buildDust() {
  const positions = [],
    colors = []
  for (let i = 0; i < 1600; i++) {
    const r = 4 + Math.random() * 14,
      a = Math.random() * Math.PI * 2
    positions.push(
      Math.cos(a) * r,
      -1.7 + Math.pow(Math.random(), 2) * 10,
      Math.sin(a) * r
    )
    const c = i % 6 === 0 ? new THREE.Color('#8ed8c9') : gold
    colors.push(c.r, c.g, c.b)
  }
  const g = new THREE.BufferGeometry()
  g.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
  g.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3))
  dust = new THREE.Points(
    g,
    new THREE.PointsMaterial({
      vertexColors: true,
      size: 0.025,
      transparent: true,
      opacity: 0.42,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    })
  )
  scene.add(dust)
}
const packetGeo = new THREE.BoxGeometry(0.1, 0.1, 0.27),
  packetMaterials = platforms.map(
    (p) =>
      new THREE.MeshBasicMaterial({
        color: new THREE.Color(p.color).multiplyScalar(1.7),
      })
  )
function spawnPacket(index, offset = 0, ambient = false) {
  if (packets.length >= 240) return
  const obj = mesh(packetGeo, packetMaterials[index])
  packets.push({
    obj,
    index,
    t: offset,
    velocity: ambient ? 0.13 : 0.22 + Math.random() * 0.08,
    ambient,
  })
}
function ripple(position, color, size = 1) {
  const obj = ring(0.28, 0.018, color)
  obj.position.copy(position)
  ripples.push({ obj, age: 0, size })
}
function burst(position, color, count = 32) {
  const geo = new THREE.BufferGeometry(),
    ps = [],
    vs = []
  for (let k = 0; k < count; k++) {
    ps.push(position.x, position.y, position.z)
    vs.push(
      (Math.random() - 0.5) * 3,
      Math.random() * 2 + 0.3,
      (Math.random() - 0.5) * 3
    )
  }
  geo.setAttribute('position', new THREE.Float32BufferAttribute(ps, 3))
  const mat = new THREE.PointsMaterial({
    color,
    size: 0.06,
    transparent: true,
    opacity: 1,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  })
  const obj = new THREE.Points(geo, mat)
  scene.add(obj)
  sparks.push({ obj, vs, age: 0 })
}
function launch() {
  if (paused) {
    toast('模拟已暂停，继续后即可发射')
    return
  }
  if (elapsed - lastLaunch < 0.4) return
  combo = elapsed - lastLaunch < 4 ? Math.min(combo + 1, 8) : 1
  lastLaunch = elapsed
  const index = selected
  for (let k = 0; k < 8 + combo * 2; k++) spawnPacket(index, -k * 0.035)
  activated.add(index)
  document
    .querySelectorAll('.goal-dots i')
    .forEach((e, i) => e.classList.toggle('lit', activated.has(i)))
  $(`#done-${index}`).textContent = '✓'
  $('#goal-count').textContent = `${activated.size} / 7`
  ripple(new THREE.Vector3(0, -1.5, 0), platforms[index].color, 1.5)
  burst(new THREE.Vector3(0, 2, 0), platforms[index].color, 24)
  chime(330 + index * 55 + combo * 22)
  if (activated.size === 7 && completionTime < 0) {
    completionTime = elapsed
    $('#goal-title').textContent = '世界已连接 · 继续创造'
    $('#goal-copy').innerHTML =
      '七站点亮，全球共振。<br><button id="replay" style="padding:0;color:#edc58d;font-size:inherit;text-decoration:underline;text-underline-offset:3px">重新启航 ↗</button>'
    $('#replay').onclick = restart
    toast('全球共振！七大平台全部点亮 ✦')
    platforms.forEach((p, i) => {
      for (let k = 0; k < 7; k++) spawnPacket(i, -k * 0.06)
      burst(nodes[i].position, p.color, 35)
    })
    ripple(new THREE.Vector3(0, -1.5, 0), '#ffe7aa', 6)
  } else if (combo >= 3) toast(`增长共振 ×${combo} · ${platforms[index].name}`)
  updateStats()
}
function restart() {
  activated.clear()
  completionTime = -100
  combo = 1
  lastLaunch = -10
  total = 0
  orderCount = 0
  platformOrders.fill(0)
  packets.forEach((p) => scene.remove(p.obj))
  packets.length = 0
  document
    .querySelectorAll('.goal-dots i')
    .forEach((e) => e.classList.remove('lit'))
  document.querySelectorAll('.done').forEach((e) => (e.textContent = ''))
  $('#goal-title').textContent = '点亮七个平台，连接世界'
  $('#goal-count').textContent = '0 / 7'
  $('#goal-copy').innerHTML = '选择平台，发射订单。<br>连续发射，触发增长共振。'
  resetCamera()
  updateStats()
  toast('新的旅程，从下一笔订单开始')
}
function selectPlatform(index, focus = true) {
  selected = index
  const p = platforms[index]
  document.documentElement.style.setProperty('--accent', p.color)
  $('#platform-index').textContent = `0${index + 1} / 07`
  $('#platform-mark').textContent = p.mark
  $('#platform-name').textContent = p.name
  $('#platform-region').textContent = p.region
  $('#platform-desc').textContent = p.desc
  $('#platform-type').textContent = p.category.split(' · ')[0]
  $('#platform-ticket').textContent = `¥${p.ticket} / 笔`
  $('#platform-orders').textContent = `${platformOrders[index]} 笔`
  document.querySelectorAll('.platform').forEach((e, i) => {
    e.classList.toggle('active', i === index)
    e.setAttribute('aria-pressed', String(i === index))
  })
  document
    .querySelectorAll('.node-label')
    .forEach((e, i) => e.classList.toggle('selected', i === index))
  if (focus && camera) {
    setOrbit(false)
    const target = nodes[index].position.clone().multiplyScalar(0.63)
    target.y = 1.5
    const dir = nodes[index].position.clone().normalize()
    const pos = target
      .clone()
      .add(dir.multiplyScalar(mobile() ? 30 : 17))
      .add(new THREE.Vector3(5, mobile() ? 24 : 15, 9))
    tweenCamera(pos, target)
    ripple(
      nodes[index].position.clone().add(new THREE.Vector3(0, -0.6, 0)),
      p.color,
      1.4
    )
    chime(440 + index * 40)
  }
}
function tweenCamera(position, target) {
  focusTween = {
    from: camera.position.clone(),
    to: position,
    targetFrom: controls.target.clone(),
    targetTo: target,
    age: 0,
  }
}
function resetCamera() {
  tweenCamera(homeCamera(), new THREE.Vector3(0, 1, 0))
  setOrbit(!reduced)
}
function setOrbit(value) {
  autoOrbit = value
  if (controls) controls.autoRotate = value
  const b = $('#orbit-toggle')
  b.classList.toggle('active', value)
  b.setAttribute('aria-pressed', String(value))
}
function updateStats() {
  $('#gmv').textContent = total.toLocaleString('en-US')
  $('#orders').textContent = orderCount.toLocaleString('en-US')
  $('#combo').textContent = `×${combo}`
  $('#platform-orders').textContent = `${platformOrders[selected]} 笔`
  $('#route-meter').style.width =
    `${35 + Math.min(platformOrders[selected], 60)}%`
}
function toggleCinema(value = !cinema) {
  cinema = value
  document.body.classList.toggle('hud-hide', value)
  $('#cinema').setAttribute('aria-pressed', String(value))
}
function openHelp(open) {
  $('#help-modal').hidden = !open
  if (open) {
    $('#close-help').focus()
  } else {
    $('#help').focus()
  }
}
function setupUI() {
  document
    .querySelectorAll('[data-select]')
    .forEach((e) =>
      e.addEventListener('click', () =>
        selectPlatform(Number(e.dataset.select))
      )
    )
  $('#launch').onclick = launch
  $('#reset').onclick = resetCamera
  $('#orbit-toggle').onclick = () => setOrbit(!autoOrbit)
  $('#pause').onclick = () => {
    paused = !paused
    $('#pause').innerHTML =
      `${svg(paused ? 'play' : 'pause')}${paused ? '继续' : '暂停'}`
    $('#pause').setAttribute('aria-label', paused ? '继续模拟' : '暂停模拟')
    $('#pause').setAttribute('aria-pressed', String(paused))
  }
  $('#speed').onclick = () => {
    if (speed === 1) {
      speed = 2
    } else if (speed === 2) {
      speed = 0.5
    } else {
      speed = 1
    }
    $('#speed').textContent = `${speed}×`
    $('#speed').setAttribute('aria-label', `模拟速度 ${speed} 倍`)
  }
  $('#audio').onclick = () => {
    audioOn = !audioOn
    $('#audio').innerHTML = svg(audioOn ? 'volume' : 'mute')
    $('#audio').setAttribute('aria-pressed', String(audioOn))
    $('#audio').setAttribute('aria-label', audioOn ? '关闭声音' : '开启声音')
    $('#audio').title = audioOn ? '关闭声音' : '开启声音'
    if (audioOn) chime()
  }
  $('#cinema').onclick = () => toggleCinema()
  $('#exit-cinema').onclick = () => toggleCinema(false)
  $('#capture').onclick = () => {
    composer.render()
    const c = document.createElement('canvas')
    c.width = renderer.domElement.width
    c.height = renderer.domElement.height
    const x = c.getContext('2d')
    x.drawImage(renderer.domElement, 0, 0)
    const s = c.width / 1440
    x.fillStyle = '#f5dfb8'
    x.font = `${22 * s}px Georgia`
    x.fillText('GMV / THE COMMERCE UNIVERSE', 40 * s, 48 * s)
    x.fillStyle = '#98a99d'
    x.font = `${11 * s}px sans-serif`
    x.fillText('中国卖家 · 全球舞台  /  交互模拟', 40 * s, c.height - 32 * s)
    const a = document.createElement('a')
    a.download = 'GMV-commerce-universe.png'
    a.href = c.toDataURL('image/png')
    a.click()
    toast('宇宙快照已保存')
  }
  $('#help').onclick = () => openHelp(true)
  $('#close-help').onclick = () => openHelp(false)
  $('#start').onclick = () => openHelp(false)
  $('#help-modal').onclick = (e) => {
    if (e.target === $('#help-modal')) openHelp(false)
  }
  document.addEventListener('keydown', (e) => {
    const modalOpen = !$('#help-modal').hidden
    if (e.key === 'Escape') {
      if (modalOpen) openHelp(false)
      else if (cinema) toggleCinema(false)
      else resetCamera()
      return
    }
    if (modalOpen) {
      if (e.key === 'Tab') {
        const focusable = [...$('#help-modal').querySelectorAll('button')]
        const first = focusable[0],
          last = focusable.at(-1)
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault()
          last.focus()
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault()
          first.focus()
        }
      }
      return
    }
    if (e.repeat || /INPUT|TEXTAREA|SELECT/.test(e.target.tagName)) return
    if (e.code === 'Space' && e.target.tagName !== 'BUTTON') {
      e.preventDefault()
      launch()
    }
    if (e.key.toLowerCase() === 'r') resetCamera()
    if (e.key.toLowerCase() === 'h') toggleCinema()
  })
  const canvas = $('#world')
  canvas.addEventListener('pointerdown', (e) =>
    activePointers.set(e.pointerId, {
      x: e.clientX,
      y: e.clientY,
      moved: false,
    })
  )
  canvas.addEventListener('pointermove', (e) => {
    const down = activePointers.get(e.pointerId)
    if (down && Math.hypot(e.clientX - down.x, e.clientY - down.y) > 6) {
      down.moved = true
    }
    pointer.set(
      (e.clientX / innerWidth) * 2 - 1,
      (-e.clientY / innerHeight) * 2 + 1
    )
    raycaster.setFromCamera(pointer, camera)
    if (raycaster.intersectObjects(hoverables, false).length) {
      canvas.style.cursor = 'pointer'
    } else if (down) {
      canvas.style.cursor = 'grabbing'
    } else {
      canvas.style.cursor = 'grab'
    }
  })
  canvas.addEventListener('pointerup', (e) => {
    const down = activePointers.get(e.pointerId),
      single = activePointers.size === 1
    activePointers.delete(e.pointerId)
    if (!down || down.moved || !single) return
    pointer.set(
      (e.clientX / innerWidth) * 2 - 1,
      (-e.clientY / innerHeight) * 2 + 1
    )
    raycaster.setFromCamera(pointer, camera)
    const hit = raycaster.intersectObjects(hoverables, false)[0]
    if (hit) selectPlatform(hit.object.userData.index)
  })
  canvas.addEventListener('pointercancel', (e) =>
    activePointers.delete(e.pointerId)
  )
  document.addEventListener('visibilitychange', () => {
    lastFrame = 0
  })
}
function resize() {
  if (!renderer) return
  labelHalves = nodeLabels.map((el) => el.offsetWidth / 2 + 10)
  camera.aspect = innerWidth / innerHeight
  camera.updateProjectionMatrix()
  renderer.setPixelRatio(Math.min(devicePixelRatio, mobile() ? 1.5 : 1.7))
  renderer.setSize(innerWidth, innerHeight)
  composer.setPixelRatio(renderer.getPixelRatio())
  composer.setSize(innerWidth, innerHeight)
  bloom.setSize(Math.floor(innerWidth * 0.65), Math.floor(innerHeight * 0.65))
  if (
    !focusTween &&
    controls.target.distanceTo(new THREE.Vector3(0, 1, 0)) < 0.2
  ) {
    camera.position.copy(homeCamera())
  }
}
let fpsTime = 0,
  fpsFrames = 0
function animate(ms) {
  if (document.hidden) {
    lastFrame = 0
    return
  }
  const dt = lastFrame ? Math.min((ms - lastFrame) / 1000, 0.06) : 0.016
  lastFrame = ms
  const delta = paused ? 0 : dt * speed
  elapsed += delta
  renderTime += dt
  stats.frames++
  fpsFrames++
  fpsTime += dt
  if (fpsTime > 1) {
    stats.fps = Math.round(fpsFrames / fpsTime)
    fpsTime = 0
    fpsFrames = 0
  }
  if (focusTween) {
    focusTween.age += dt
    const t = Math.min(focusTween.age / 1.5, 1),
      ease = 1 - Math.pow(1 - t, 3)
    camera.position.lerpVectors(focusTween.from, focusTween.to, ease)
    controls.target.lerpVectors(
      focusTween.targetFrom,
      focusTween.targetTo,
      ease
    )
    if (t >= 1) focusTween = null
  }
  controls.update(dt)
  if (delta) {
    earth.rotation.y += delta * 0.045
    rotors.forEach((r, i) => {
      if (i < 3) r.rotation.z += delta * (i % 2 ? -0.07 : 0.09)
      else r.rotation.y += delta * (r.userData.platform ? 0.025 : 0.04)
    })
    core.position.y = 2 + Math.sin(elapsed * 0.5) * 0.1
    dust.rotation.y += delta * 0.007
    nodes.forEach((n, i) => {
      n.position.y = platforms[i].height + Math.sin(elapsed * 0.65 + i) * 0.13
      lights[i].intensity = selected === i ? 15 : 6
    })
    autoTime += delta
    if (autoTime > 1.0) {
      autoTime = 0
      spawnPacket(Math.floor(Math.random() * 7), 0, true)
    }
    for (let i = packets.length - 1; i >= 0; i--) {
      const p = packets[i]
      p.t += delta * p.velocity
      if (p.t < 0) {
        p.obj.visible = false
        continue
      }
      p.obj.visible = true
      if (p.t >= 1) {
        scene.remove(p.obj)
        packets.splice(i, 1)
        orderCount++
        total += platforms[p.index].ticket
        platformOrders[p.index]++
        if (!p.ambient) {
          ripple(
            nodes[p.index].position.clone().add(new THREE.Vector3(0, -0.5, 0)),
            platforms[p.index].color,
            0.6
          )
          if (Math.random() < 0.22) {
            burst(nodes[p.index].position, platforms[p.index].color, 12)
          }
        }
        continue
      }
      const point = curves[p.index].getPoint(p.t),
        tangent = curves[p.index].getTangent(p.t)
      p.obj.position.copy(point)
      p.obj.quaternion.setFromUnitVectors(new THREE.Vector3(0, 0, 1), tangent)
      p.obj.scale.setScalar(p.ambient ? 0.8 : 1.35)
    }
    for (let i = ripples.length - 1; i >= 0; i--) {
      const r = ripples[i]
      r.age += delta
      const s = 1 + r.age * 3 * r.size
      r.obj.scale.setScalar(s)
      r.obj.material.transparent = true
      r.obj.material.opacity = Math.max(0, 1 - r.age / 1.7) * 0.55
      if (r.age > 1.7) {
        scene.remove(r.obj)
        r.obj.geometry.dispose()
        r.obj.material.dispose()
        ripples.splice(i, 1)
      }
    }
    for (let i = sparks.length - 1; i >= 0; i--) {
      const s = sparks[i]
      s.age += delta
      const pos = s.obj.geometry.attributes.position
      for (let k = 0; k < pos.count; k++) {
        pos.array[k * 3] += s.vs[k * 3] * delta
        pos.array[k * 3 + 1] += s.vs[k * 3 + 1] * delta
        pos.array[k * 3 + 2] += s.vs[k * 3 + 2] * delta
        s.vs[k * 3 + 1] -= delta * 0.6
      }
      pos.needsUpdate = true
      s.obj.material.opacity = 1 - s.age / 1.5
      if (s.age > 1.5) {
        scene.remove(s.obj)
        s.obj.geometry.dispose()
        s.obj.material.dispose()
        sparks.splice(i, 1)
      }
    }
    if (elapsed - lastLaunch > 4 && combo !== 1) combo = 1
    if (Math.floor(elapsed * 8) !== Math.floor((elapsed - delta) * 8)) {
      updateStats()
    }
  }
  nodes.forEach((node, i) => {
    tmp
      .copy(node.position)
      .add(new THREE.Vector3(0, 1.95, 0))
      .project(camera)
    const x = (tmp.x * 0.5 + 0.5) * innerWidth,
      y = (-tmp.y * 0.5 + 0.5) * innerHeight
    const el = nodeLabels[i],
      half = labelHalves[i],
      visible =
        tmp.z < 1 &&
        x > 0 &&
        x < innerWidth &&
        y > 80 &&
        y < innerHeight - (mobile() ? 210 : 160)
    el.style.left = `${Math.max(half, Math.min(innerWidth - half, x))}px`
    el.style.top = `${y}px`
    el.style.opacity = visible ? '1' : '0'
    el.style.visibility = visible ? 'visible' : 'hidden'
  })
  tmp.set(0, -1, 0).project(camera)
  const coreLabel = $('.core-label')
  coreLabel.style.left = `${(tmp.x * 0.5 + 0.5) * innerWidth}px`
  coreLabel.style.top = `${(-tmp.y * 0.5 + 0.5) * innerHeight}px`
  coreLabel.style.transform = 'translate(-50%,0)'
  coreLabel.style.opacity = mobile() ? '0' : '.7'
  bloom.strength =
    0.55 +
    (completionTime > 0 && elapsed - completionTime < 4
      ? Math.sin(((elapsed - completionTime) / 4) * Math.PI) * 0.65
      : 0)
  renderer.info.reset()
  composer.render()
}

try {
  init()
} catch (e) {
  fatal(e)
}
