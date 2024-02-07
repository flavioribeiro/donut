// just to avoid adding dup messages
window.metadataMessages = {}

window.startSession = () => {
  let srtHost = document.getElementById('srt-host').value;
  let srtPort = document.getElementById('srt-port').value;
  let srtStreamId = document.getElementById('srt-stream-id').value;

  setupWebRTC((pc, offer) => {
    let srtFullAddress = JSON.stringify({
      "srtHost": srtHost,
      "srtPort": srtPort,
      "srtStreamId": srtStreamId,
      offer
    });

    // sending localSDP,SRT params, and fetching remote SDP
    fetchRemoteDescription(srtFullAddress).then(remoteOffer => {
      log("receiving remote sdp offer: " + JSON.stringify(remoteOffer));

      if (remoteOffer === undefined) {
        log("error while fetching remote");
        return;
      }
      pc.setRemoteDescription(remoteOffer);
    }).catch(log, "error");
  });
}

const setupWebRTC = (setRemoteSDPfn) => {
  log("setting up web rtc");
  const pc = new RTCPeerConnection({
    iceServers: [{
      urls: 'stun:stun.l.google.com:19302'
    }]
  });

  // offer to (only) receive 1 audio, and 1 video track
  pc.addTransceiver('video', {
    direction: 'recvonly'
  });
  pc.addTransceiver('audio', {
    direction: 'recvonly'
  });

  // once a track arrives, add it to the remoteVideos div
  // with auto play.
  pc.ontrack = function (event) {
    log("ontrack : " + event.track.kind + " label " + event.track.label);

    const el = document.createElement(event.track.kind);
    el.srcObject = event.streams[0];
    el.autoplay = true
    el.controls = true;
    el.width = "640";
    el.height = "360";

    document.getElementById('remoteVideos').appendChild(el);
  }

  pc.createDataChannel('metadata');
  // once the metadata arrives, add it to the metadata div
  pc.ondatachannel = (e) => {
    log("ondatachannel: " + JSON.stringify(e));

    e.channel.onmessage = (event) => {
      let msg = JSON.parse(event.data)
      if (msg.Message in metadataMessages) {
        // avoid logging dup messages
        return;
      }

      const el = document.createElement("p")
      el.innerText = msg.Type.padEnd(8, ' ') + ": " + msg.Message

      let metadata = document.getElementById('metadata');
      metadata.insertBefore(el, metadata.firstChild);
      metadataMessages[msg.Message] = true;
    };
  };

  pc.oniceconnectionstatechange = e => log("ice state change: " + pc.iceConnectionState);
  pc.onicegatheringstatechange = e => log("gathering state change: " + pc.iceGatheringState);
  pc.onsignalingstatechange = e => log("signaling state change: " + pc.signalingState);

  // creating a local sdp offer
  pc.createOffer()
    .then(offer => {
      pc.setLocalDescription(offer);
      setRemoteSDPfn(pc, offer);
    }).catch(log, "error");
}

const fetchRemoteDescription = async (bodyRequest) => {
  log("requesting remote sdp offer for: " + bodyRequest)

  const res = await fetch('/doSignaling', {
    method: 'post',
    headers: {
      'Accept': 'application/json, text/plain, */*',
      'Content-Type': 'application/json'
    },
    body: bodyRequest
  });

  if (res.status !== 200) {
    res.text().then(err => {
      log(err, "error");
      window.alert(err);
    });
    return;
  }

  return res.json();
}

const formattedNow = () => {
  let now = new Date();
  let minutes = now.getMinutes().toString().padStart(2, '0');
  let seconds = now.getSeconds().toString().padStart(2, '0');
  let ms = now.getMilliseconds().toString().padStart(3, '0');
  return minutes + ":" + seconds + ":" + ms;
}

const log = (msg, level = "info") => {
  const el = document.createElement("p")
  if (level === "error") {
    el.style = "color: red;background-color: yellow;";
  }

  el.innerText = "[[" + level.toUpperCase().padEnd(5, ' ') + "]] " + formattedNow() + " : " + msg

  let logEl = document.getElementById('log');
  logEl.insertBefore(el, logEl.firstChild);
}