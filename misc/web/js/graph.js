var nodes = [];
var edges = [];
var netedges = [];
var force, svgNodes, svgEdges;
var canvasWidth = window.innerWidth - 100;
var canvasHeight = window.innerHeight - 100;
var keepFit = true;
var renderView = "all";
var DENSITY_MAX = 10000;
var UPDATE_PERIOD = 1500;
var needReforce = false;
var UNCONNECTED = -1;
var textYoffset = 2;

var svgZoom = d3.behavior.zoom();
var svg = d3.select("#graph").attr("width", canvasWidth).attr("height", canvasHeight).call(svgZoom.on("zoom",zoom)).append("g");
var svgE = svg.append("g");
var svgN = svg.append("g");
var svgNE = svg.append("g");

var btnFit = d3.select("#btnfit");
var searchBox = d3.select("#searchbox");
var spinner = d3.select("#spinner").attr("class", "loader");


var config = {
	settleFactor: 10,
	settleMax: 10000,
	gravity: 0.3,
	linkstrength: 1,
	linkdistance: 10,
	charge: -1000,
	friction: 0.8,
	theta: 1,
};

var defaultIconWidth  = 30;
var defaultIconHeight = 30;

var iconLinks = {
	default: "/images/desktop.png",

	// system
	desktop: "/images/desktop.png",
	laptop: "/images/laptop.png",
	mobile: "/images/mobile.png",
	server: "/images/server.png",
	printer: "/images/printer.png",

	// roygbv
	redcircle: "/images/red-circle.png",
	orangecircle: "/images/orange-circle.png",
	yellowcircle: "/images/yellow-circle.png",
	greencircle: "/images/green-circle.png",
	bluecircle: "/images/blue-circle.png",
	violetcircle: "/images/violet-circle.png",

	redsquare: "/images/red-square.png",
	greensquare: "/images/green-square.png",
	orangesquare: "/images/orange-square.png",

	highlight: "/images/orange-circle.png",
	highlightSearch: "/images/green-circle.png",
};

function zoom(t,s) {
	if (t == null) {
		svg.attr("transform", "translate(" + d3.event.translate + ")scale(" + d3.event.scale + ")");
		keepFit = false;
	} else {
		svg.attr("transform", "translate(" + t[0] + "," + t[1] + ")scale(" + s + ")");
		svgZoom.scale(s);
		svgZoom.translate(t);
	}
	checkDensity();
	tick();
};

// checkDensity looks at how many nodes are currently being rendered on screen.
// If it is above a threshold, we render a network view only.
function checkDensity() {
	if (svgNodes == null) {
		return;
	}
	count = 0;
	svgNodes.each(function(d) {
		if (onCanvas(d.x, d.y)) {
			count++;
		}
	});
	oldView = renderView;
	if (count > DENSITY_MAX) {
		networkView();
	} else {
		allView();
	}
}

function networkView() {
	svgNodes.attr("visibility", function(d) { if (isNode(d)) { return "hidden"; } else { return "visible"; } });
	svgEdges.attr("visibility", "hidden");
	svgNetEdges.attr("visibility", "visible");

}

function allView() {
	svgNodes.attr("visibility", "visible");
	svgEdges.attr("visibility", "visible");
	svgNetEdges.attr("visibility", "hidden");
}

function fit() {
	minx = 0;
	miny = 0;
	maxx = 0;
	maxy = 0;
	for (var i = 0; i < nodes.length; i++) {
		if (i == 0) {
			minx = nodes[i].x;
			maxx = nodes[i].x;
			miny = nodes[i].y;
			maxy = nodes[i].y;
			continue;
		}
		if (nodes[i].x < minx) {
			minx = nodes[i].x;
		}
		if (nodes[i].x > maxx) {
			maxx = nodes[i].x;
		}
		if (nodes[i].y < miny) {
			miny = nodes[i].y;
		}
		if (nodes[i].y > maxy) {
			maxy = nodes[i].y;
		}
	}
	sx = canvasWidth / (maxx - minx);
	sy = canvasHeight / (maxy - miny);
	s = Math.min(sx,sy) - .1;
	x = (canvasWidth / 2);
	y = (canvasHeight / 2);

	if (nodes.length == 1) {
		s = 1;
	}

	zoom([x,y], s);
	keepFit = true;
}

function copy(src, dst) {
	for (var p in src) {
		if (src.hasOwnProperty(p)) {
			dst[p] = src[p];
		}
	}
}

function update(error, json) {
	needReforce = false;

	if (error) return console.warn(error);

	for (var i = 0; i < nodes.length; i++) {
		nodes[i].keep = false;
	}
	for (var i = 0; i < edges.length; i++) {
		edges[i].keep = false;
	}

	if (json != null) {
		for (var i = 0; i < json.length; i++) {
			nnid = name(json[i]);
			found = false;

			// search for and update existing nodes
			for (var j = 0; j < nodes.length; j++) {
				onid = name(nodes[j]);
				if (nnid == onid) {
					copy(json[i], nodes[j]);
					nodes[j].keep = true;
					found = true;
					if (isNode(nodes[j])) {
						resolveEdges(j);
					}
					break;
				}
			}

			// insert new nodes
			if (!found) {
				needReforce = true;
				nodes.push(json[i]);
				nodes[nodes.length-1].keep = true;
				if (isNode(nodes[nodes.length-1])) {
					resolveEdges(nodes.length-1);
				}
			}
		}
	}

	// trim nodes
	for (var i = 0; i < nodes.length; i++) {
		if (!nodes[i].keep) {
			needReforce = true;
			nodes.splice(i,1);
			i--;
			// fix edge indexes above i
			for (var j = 0; j < edges.length; j++) {
				if (edges[j].sid >= i) {
					edges[j].sid--;
					edges[j].source = edges[j].sid;
				}
				if (edges[j].tid >= i) {
					edges[j].tid--;
					edges[j].target = edges[j].tid;
				}
			}
		}
	}

	// remove dangling edges
	for (var i = 0; i < edges.length; i++) {
		if (!edges[i].keep) {
			needReforce = true;
			edges.splice(i,1);
			i--;
		}
	}

	// simply rebuild netedges
	netedges = [];
	for (var i = 0; i < edges.length; i++) {
		var near, far;
		if (!isNode(nodes[edges[i].sid])) {
			near = edges[i].sid;
			far = edges[i].tid;
		} else {
			near = edges[i].tid;
			far = edges[i].sid;
		}
		// draw an edge to all switches that are on the far side
		for (var j = 0; j < edges.length; j++) {
			if (edges[j].sid == far) {
				netedges.push({ sid: near, tid: edges[j].tid });
			} else if (edges[j].tid == far) {
				netedges.push({ sid: near, tid: edges[j].sid });
			}
		}
	}

	redraw();
	checkDensity();
	if (needReforce) {
		force.start();
	}
	if (keepFit) {
		fit();
	}
	needReforce = false;

	//setTimeout(checkUpdate, UPDATE_PERIOD);
	//d3.timer(checkUpdate, UPDATE_PERIOD);
}

// patch up edge indices with new node information. resolveEdges is always
// invoked before nodes are spliced from the nodes array, so existing indices
// should always be sane.
function resolveEdges(idx) {
	if (nodes[idx].Edges == null) {
		return;
	}
	for (var i = 0; i < nodes[idx].Edges.length; i++) {
		if (nodes[idx].Edges[i].N == UNCONNECTED) {
			continue;
		}
		var foundNode = false;
		for (var j = 0; j < nodes.length; j++) {
			if (!isNode(nodes[j]) && nodes[j].NID == nodes[idx].Edges[i].N) {
				foundNode = true;
				nodes[j].keep = true;

				// edge should be source: idx, target: jj
				// if we don't already have this edge, add it
				foundEdge = false;
				for (k = 0; k < edges.length; k++) {
					if (edges[k].sid == idx && edges[k].tid == j) {
						foundEdge = true;
						edges[k].keep = true;
						break;
					}
				}
				if (!foundEdge) {
					needReforce = true;
					edges.push({ source: idx, target: j, sid: idx, tid: j, keep: true });
				}
				break;
			}
		}
		if (!foundNode) {
			needReforce = true;
			nodes.push({ NID: nodes[idx].Edges[i].N, keep: true });
			edges.push({ source: idx, target: nodes.length-1, sid: idx, tid: nodes.length-1, keep: true });
		}
	}
}

function isNode2(n) {
	if ("shape" in n) {
		return true;
	}
	return false;
}

function redraw() {
	force.nodes(nodes);
	force.links(edges);

	var filteredEdges = edges.filter(function(edge){
		if (nodes[edge.target] && nodes[edge.target].Endpoints){
			if (nodes[edge.target].Endpoints.length < 2)
				return false;
		}
		return true;
	});
	svgEdges = svgE.selectAll("line").data(filteredEdges);
	svgEdges.enter().insert("line").transition();
	svgEdges.exit().remove();

	svgNetEdges = svgNE.selectAll("line").data(netedges);
	svgNetEdges.enter().insert("line").transition();
	svgNetEdges.exit().remove();

	var endpoints = nodes.filter(function(node){
		return !!node.D;
	});
	svgNodes = svgN.selectAll("g").data(endpoints);
	newNodes = svgNodes.enter().insert("g")
		.on("mouseover", function(d) {
			if (!isNode(d)) {
				return;
			}
			y = 10;
			x = 10;

			d3.select("#highlight-" + d.NID).attr("style", "");
			info(d);
			d3.select("#info").classed("hidden", false);
		})
		.on("mouseout", function(d, index) {
			d3.select("#info").classed("hidden", true);
			d3.select("#highlight-" + d.NID).attr("style", "display: none;");
		});

	// Add custom shapes to nodes, defaults to circle
	newNodes.each(addShape);

	//newNodes.insert("old-circle");
	//newNodes.insert("text");
	svgNodes.exit().transition().remove();

	// endpoint/vlan css class
	// ### i'm not sure this was ever doing anything -cdshann
	svgN.selectAll("old-circle")
		.attr("class", function(d) {
			if (!isNode(d)) {
				return "vlan";
			} else {
				return "endpoint";
			}
		});

	svgN.selectAll("text").text(function(d) {
		if (isNode(d)) {
			if (d.D.hasOwnProperty("text")) {
				return d.D["text"];
			}
		}
		return "";
	});
}

function getImage(imageName) {
	var img = {
		link: iconLinks['default'],
		width: defaultIconWidth,
		height: defaultIconHeight,
	}

	if (imageName in iconLinks) {
		img.link = iconLinks[imageName];
	}

	// hard-coded check for circle, for hightlights
	if (img.link.indexOf("circle") !== -1) {
		img.width += 8;
		img.height += 8;
	}

	return img;
}

function addShape(n) {
	if (!isNode(n)) {
		return;
	}
	if ("shape" in n.D && n.D["shape"] in iconLinks) {
		var imageName = n.D["shape"];
	} else {
		// Default image
		var imageName = "default";
	}

	var highlightSearch = getImage("highlightSearch");
	var highlight = getImage("highlight");
	var img = getImage(imageName);

	d3.select(this).insert("image")
		.attr("xlink:href", highlightSearch.link).attr("width", highlightSearch.width).attr("height", highlightSearch.height)
		.attr("style", "display: none;").attr("id", "highlight-search-" + n.NID).attr("class", "highlight-search");
	d3.select(this).insert("image")
		.attr("xlink:href", highlight.link).attr("width", highlightSearch.width).attr("height", highlightSearch.height)
		.attr("style", "display: none;").attr("id", "highlight-" + n.NID).attr("class", "highlight");
	d3.select(this).insert("image")
		.attr("xlink:href", img.link).attr("width", img.width).attr("height", img.height);
}

function resize() {
	canvasWidth = window.innerWidth - 100;
	canvasHeight = window.innerHeight - 100;
	d3.select("#graph").attr("width", canvasWidth).attr("height", canvasHeight);
	if (keepFit) {
		fit();
	}
};
window.addEventListener("resize", resize);

// return a unique name for a vm
// TODO: should be UUID?
function name(n) {
	return n.NID;
}

function isNode(n) {
	if (n.hasOwnProperty("Edges")) {
		return true;
	}
	return false;
}

function checkUpdate() {
	d3.json("nodes" + '?nocache=' + (new Date()).getTime(), update);
};

function start() {
	while (force.alpha() > 0) {
		force.tick();
	}
	fit();
	tick();
	checkDensity();
	spinner.attr("style","display: none;");
	force.on("tick", tick);
	force.on("start", force.start);
};

function tick() {
	svgEdges.attr("x1", function(d) { return d.source.x; })
		.attr("y1", function(d) { return d.source.y; })
		.attr("x2", function(d) { return d.target.x; })
		.attr("y2", function(d) { return d.target.y; });
	svgNetEdges.attr("x1", function(d) { return nodes[d.sid].x; })
		.attr("y1", function(d) { return nodes[d.sid].y; })
		.attr("x2", function(d) { return nodes[d.tid].x; })
		.attr("y2", function(d) { return nodes[d.tid].y; });
	svgN.selectAll("image")
		.attr("x", function(d) {
			if ("shape" in d.D){
				var img = getImage(d.D["shape"]);
			} else {
				var img = getImage("default");
			}
			return  d.x - (img.width / 2);
		}).attr("y", function(d) {
			if ("shape" in d.D){
				var img = getImage(d.D["shape"]);
			} else {
				var img = getImage("default");
			}
			return d.y - (img.height / 2);
		});
	d3.selectAll(".highlight-search")
		.attr("x", function(d) {
			var img = getImage("highlightSearch");
			return  d.x - (img.width / 2);
		}).attr("y", function(d) {
			var img = getImage("highlightSearch");
			return d.y - (img.height / 2);
		});
	d3.selectAll(".highlight")
		.attr("x", function(d) {
			var img = getImage("highlight");
			return  d.x - (img.width / 2);
		}).attr("y", function(d) {
			var img = getImage("highlight");
			return d.y - (img.height / 2);
		});
	svgN.selectAll("text").attr("x", function(d) { return d.x; })
		.attr("y", function(d) { return d.y + textYoffset; });
}

function onCanvas(x, y) {
	s = svgZoom.scale();
	t = svgZoom.translate();
	x = x*s+ t[0];
	y = y*s + t[1];
	if (x > 0 && y > 0 && x < canvasWidth && y < canvasHeight) {
		return true;
	}
	return false;
}

btnFit.on("click", function() {
	fit();
});

function searchNodes(value) {
	if (value) {
		$.ajax({url: "/endpoints/"+value, success: function(result){
			d3.selectAll(".highlight-search").attr("style", "display: none;");
			if (result) {
				result.forEach(function(n){
					d3.select("#highlight-search-" + n.NID).attr("style", "");
				})
			}
		}});
	} else {
		d3.selectAll(".highlight-search").attr("style", "display: none;");
	}
}

searchBox.on("change", function() {
	searchNodes(this.value);
});

force = d3.layout.force()
	.nodes(nodes)
	.links(edges)
	.linkDistance(config.linkdistance)
	.charge(config.charge)
	.linkStrength(config.linkstrength)
	.gravity(config.gravity)
	.friction(config.friction)
	.theta(config.theta)
	.on("start", start)
	.on("end", function() {
		if (keepFit) {
			fit();
		}
	});

checkUpdate();

function info(n) {
	// get rid of old data
	d3.select("#info")
		.select("#value")
		.remove();

	// and recreate it
	f = d3.select("#info")
		.insert("span")
		.attr("id", "value");

	if (isNode(n)) {
		if (n.D.hasOwnProperty("image") && n.D["image"]) {
			f.append("img").attr("src", "/image/" + n.D["image"]);
		}
		if (n.D.hasOwnProperty("picture") && n.D["picture"]) {
			f.append("img").attr("src", "data:image/jpeg;base64," + n.D["picture"]).attr("height", "200px");
		}
		var table = f.append('table');
		var node = [];

		if (n.D.hasOwnProperty("hostname") && n.D["hostname"]) {
			var hostnames = n.D["hostname"].split(",").sort();
			if (hostnames.length > 1){
				node.push(["Hostnames", hostnames[0]])
				hostnames.slice(1).forEach(function(hostname){
					node.push(["", hostname]);
				});
			} else {
				node.push(["Hostname", hostnames[0]])
			}
		}
		if (n.D.hasOwnProperty("os") && n.D["os"]) {
			var os = n.D["os"].split(":");
			if (os[0] == "s") {
				os = os.slice(1)
			}
			if (os[0] == "g") {
				os = os.slice(1)
			}
			if (os[0] == "unix") {
				os = os.slice(1)
			}
			if (os[0] == "win") {
				os = os.slice(1)
			}
			os = os.join(" ")
			node.push(["OS", os]);
		}
		if (n.D.hasOwnProperty("description") && n.D["description"]) {
			var description = n.D["description"].split("\n");
			node.push(["Description", description[0]]);
			description.slice(1).forEach(function(line){
				node.push(["", line]);
			});
		}
		if (n.D.hasOwnProperty("ports") && n.D["ports"]) {
			var ports = n.D["ports"].split(",").sort(function(a,b){return a-b});
			node.push(["Ports", ports.join(", ")])
		}
		if (n.D.hasOwnProperty("nameserver") && n.D["nameserver"]) {
			var nameservers = n.D["nameserver"].split(",").sort();
			if (nameservers.length > 1){
				node.push(["Nameservers", nameservers[0]])
				nameservers.slice(1).forEach(function(nameserver){
					node.push(["", nameserver]);
				});
			} else {
				node.push(["Nameserver", nameservers[0]])
			}
		}
		if (n.D.hasOwnProperty("name") && n.D["name"]) {
			node.push(["Name", n.D["name"]]);
		}
		if (n.D.hasOwnProperty("uuid") && n.D["uuid"]) {
			node.push(["UUID", n.D["uuid"]]);
		}

		table.append('thead').append('tr')
			.selectAll('th').data(["", ""])
			.enter()
			.append('th')
			.text(function (column) { return column; });

		table.append('tbody')
			.selectAll("tr")
				.data(node).enter()
				.append("tr")
			.selectAll("td")
				.data(function(d) { return d; }).enter()
				.append("td")
				.text(function(d) { return d; });

		f.selectAll("img").attr("style", "display: block;margin-left: auto;margin-right: auto;")

		var edgesTable = f.append('table');
		var edgesNode = [];
		if (n.Edges.length > 0) {
			for (i=0; i<n.Edges.length; i++) {
				edgesNode.push(["Edge " + i, "Network:", n.Edges[i].N]);
				if (n.Edges[i].D.hasOwnProperty("ip")) {
					edgesNode.push(["", "IP:", n.Edges[i].D["ip"]]);
				}
				if (n.Edges[i].D.hasOwnProperty("ip6")) {
					edgesNode.push(["", "IPv6:", n.Edges[i].D["ip6"]]);
				}
			}
		}

		edgesTable.append('thead').append('tr')
			.selectAll('th').data(["", "", ""])
			.enter()
			.append('th')
			.text(function (column) { return column; });

		edgesTable.append('tbody')
			.selectAll("tr")
				.data(edgesNode).enter()
				.append("tr")
			.selectAll("td")
				.data(function(d) { return d; }).enter()
				.append("td")
				.text(function(d) { return d; });
	}
}
