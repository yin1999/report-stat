'use strict';

const select = document.querySelector('#class-select')
const classList = ['全部', '物联网18_1', '物联网18_2', '物联网18_3', '电科18_1', '电科18_2', '自动化18_1', '自动化18_2', '自动化18_3', '计算机18_1', '计算机18_2', '计算机18_3', '通信18_1', '通信18_2', '通信18_3']

for (const className of classList) {
	const option = document.createElement('option')
	option.value = className
	option.innerText = className
	select.appendChild(option)
}

const cookieMaxAge = 5 * 24 * 3600
const value = getCookie('class')
if (value) {
	setCookie('class', value, cookieMaxAge)
	select.value = value
}

const img = document.querySelector('#img')

if (navigator.userAgent.includes('QQ/') || navigator.userAgent.includes('MicroMessenger')) { //Mobile QQ or Wechat
	let status = null
	function setImage(name) {
		img.src = `image/${name}.webp?${status.lastModified}` // avoid cache
	}
	fetch('image/status.json')
		.then(res => res.json())
		.then(res => {
			status = res
			setImage(select.value)
			showMetaData(status.lastModified)
			setNumber(status.remains[select.value])
		})
		.catch(console.error)
	select.addEventListener('change', e => {
		const className = e.target.value
		setCookie('class', className, cookieMaxAge)
		setNumber(status.remains[className])
		setImage(className)
	})
} else {
	let rawData = []
	fetch('data.json', { credentials: "omit" })
		.then(res => res.json())
		.then(res => {
			const data = res.formData || []
			const className = select.value
			rawData = data
			const tableData = filter(rawData, className)
			setNumber(tableData.length)
			renderTable(tableData, className === "全部")
			showMetaData(res.lastModified)
		})
		.catch(console.error)

	select.addEventListener('change', e => {
		const className = e.target.value
		setCookie('class', className, cookieMaxAge)
		const tableData = filter(rawData, className)
		setNumber(tableData.length)
		renderTable(tableData, className === "全部")
	})
}

function filter(data, classname) {
	if (classname === "全部") {
		return data
	}
	let start = 0
	for (start in data) {
		if (data[start][4] == classname) {
			break
		}
	}
	if (start === data.length) {
		return []
	}
	let end = data.length - 1
	while (data[end][4] !== classname) {
		end--
	}
	return data.slice(start, end + 1)
}

function setNumber(remain) {
	document.querySelector('#number').innerText = remain
}

function showMetaData(lastModified) {
	const lm = document.querySelector('#lastModified')
	const date = new Date(lastModified * 1000)
	lm.dateTime = date.toISOString()
	lm.innerText = date.toLocaleString('zh-CN', { dateStyle: 'short', timeStyle: 'medium', hourCycle: 'h24' })
	document.querySelector('#metadata').hidden = false
}

function getCookie(key) {
	key = encodeURIComponent(key)
	const cookies = document.cookie.split('; ')
	for (const cookie of cookies) {
		const [k, v] = cookie.split('=')
		if (k === key) {
			return decodeURIComponent(v)
		}
	}
	return null
}

function setCookie(key, value, maxAge = undefined) {
	document.cookie = `${encodeURIComponent(key)}=${encodeURIComponent(value)}` + (maxAge !== undefined ? `;max-age=${maxAge}` : '')
}

/**
 * 
 * @param {Array<Array<string>>} tableData 
 * @param {boolean} all
 */
function renderTable(tableData, all) { // todo add classname
	const canvas = document.createElement('canvas')
	const ctx = canvas.getContext('2d')
	ctx.font = '18px bold serif'

	const idWidth = 122
	const nameWidth = 86
	const classWidth = 102
	const height = 25
	canvas.width = idWidth + nameWidth + (all ? classWidth : 0)
	canvas.height = tableData.length * height + height

	ctx.fillStyle = '#fff'
	ctx.fillRect(0, 0, canvas.width, canvas.height)
	ctx.strokeStyle = '#ccc'

	for (let i = 0; i <= canvas.height; i += height) {
		ctx.beginPath()
		ctx.moveTo(0, i)
		ctx.lineTo(canvas.width, i)
		ctx.stroke()
	}

	ctx.beginPath()
	ctx.moveTo(0, 0)
	ctx.lineTo(0, canvas.height)
	ctx.stroke()

	ctx.beginPath()
	ctx.moveTo(idWidth, 0)
	ctx.lineTo(idWidth, canvas.height)
	ctx.stroke()

	if (all) {
		const width = idWidth + nameWidth
		ctx.beginPath()
		ctx.moveTo(width, 0)
		ctx.lineTo(width, canvas.height)
		ctx.stroke()
	}

	ctx.beginPath()
	ctx.moveTo(canvas.width, 0)
	ctx.lineTo(canvas.width, canvas.height)
	ctx.stroke()

	ctx.fillStyle = '#000'
	ctx.font = '18px bold serif'
	const width1 = idWidth / 2
	const width2 = idWidth + nameWidth / 2
	const width3 = idWidth + nameWidth + classWidth / 2

	draw(ctx, width1, height - 5, '学号')
	draw(ctx, width2, height - 5, '姓名')
	all && draw(ctx, width3, height - 5, '班级')

	for (let i = 0; i < tableData.length; i++) {
		const h = (i + 2) * height - 6
		draw(ctx, width1, h, tableData[i][0])
		draw(ctx, width2, h, tableData[i][1])
		all && draw(ctx, width3, h, tableData[i][4])
	}

	canvas.toBlob(blob => {
		const url = URL.createObjectURL(blob)
		if (img.src !== '') {
			const src = img.src
			img.src = ''
			URL.revokeObjectURL(src)
		}
		img.src = url
	}, 'image/webp')
}

function draw(ctx, x, y, text) {
	const { width } = ctx.measureText(text)
	ctx.fillText(text, x - width / 2, y)
}
