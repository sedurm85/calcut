(function() {
    'use strict';

    let wasmReady = false;
    let currentFile = null;
    let splitResults = null;

    const elements = {
        dropZone: document.getElementById('drop-zone'),
        fileInput: document.getElementById('file-input'),
        browseBtn: document.getElementById('browse-btn'),
        fileInfo: document.getElementById('file-info'),
        fileName: document.getElementById('file-name'),
        fileMeta: document.getElementById('file-meta'),
        removeFile: document.getElementById('remove-file'),
        options: document.getElementById('options'),
        sizeOptions: document.getElementById('size-options'),
        maxSize: document.getElementById('max-size'),
        prefix: document.getElementById('prefix'),
        splitBtn: document.getElementById('split-btn'),
        results: document.getElementById('results'),
        resultSummary: document.getElementById('result-summary'),
        resultList: document.getElementById('result-list'),
        downloadAll: document.getElementById('download-all'),
        resetBtn: document.getElementById('reset-btn'),
        loading: document.getElementById('loading'),
    };

    async function initWasm() {
        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch('js/ical.wasm'),
            go.importObject
        );
        go.run(result.instance);
        wasmReady = true;
    }

    function formatBytes(bytes) {
        if (bytes >= 1024 * 1024) {
            return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
        } else if (bytes >= 1024) {
            return (bytes / 1024).toFixed(1) + ' KB';
        }
        return bytes + ' bytes';
    }

    function showElement(el) {
        el.classList.remove('hidden');
    }

    function hideElement(el) {
        el.classList.add('hidden');
    }

    function handleFile(file) {
        if (!file || !file.name.endsWith('.ics')) {
            alert('.ics 파일만 지원합니다.');
            return;
        }

        currentFile = file;
        
        const reader = new FileReader();
        reader.onload = function(e) {
            currentFile.content = e.target.result;
            
            if (wasmReady && window.calcut) {
                const info = window.calcut.getInfo(currentFile.content);
                elements.fileMeta.textContent = `${formatBytes(file.size)} · ${info.events}개 이벤트`;
            } else {
                elements.fileMeta.textContent = formatBytes(file.size);
            }
        };
        reader.readAsText(file);

        elements.fileName.textContent = file.name;
        hideElement(elements.dropZone);
        showElement(elements.fileInfo);
        showElement(elements.options);
        hideElement(elements.results);
    }

    function resetUI() {
        currentFile = null;
        splitResults = null;
        elements.fileInput.value = '';
        showElement(elements.dropZone);
        hideElement(elements.fileInfo);
        hideElement(elements.options);
        hideElement(elements.results);
        hideElement(elements.loading);
    }

    function getSplitMode() {
        return document.querySelector('input[name="split-mode"]:checked').value;
    }

    function performSplit() {
        if (!currentFile || !currentFile.content) {
            alert('파일을 먼저 선택해주세요.');
            return;
        }

        if (!wasmReady || !window.calcut) {
            alert('WASM 모듈 로딩 중입니다. 잠시 후 다시 시도해주세요.');
            return;
        }

        showElement(elements.loading);
        hideElement(elements.options);

        setTimeout(function() {
            const mode = getSplitMode();
            const options = {
                mode: mode,
                maxSize: mode === 'size' ? elements.maxSize.value : '',
                prefix: elements.prefix.value.trim(),
            };

            const result = window.calcut.split(currentFile.content, options);

            hideElement(elements.loading);

            if (result.error) {
                alert('오류: ' + result.error);
                showElement(elements.options);
                return;
            }

            splitResults = result.files;
            displayResults(result);
        }, 50);
    }

    function displayResults(result) {
        elements.resultSummary.textContent = 
            `✅ ${result.totalEvents}개 이벤트 → ${result.files.length}개 파일로 분할 완료`;

        elements.resultList.innerHTML = '';
        
        result.files.forEach(function(file, index) {
            const item = document.createElement('div');
            item.className = 'result-item';
            item.innerHTML = `
                <div class="result-item-info">
                    <span class="result-item-name">${file.filename}</span>
                    <span class="result-item-meta">${formatBytes(file.size)} · ${file.events}개 이벤트</span>
                </div>
                <button type="button" class="download-btn" data-index="${index}">다운로드</button>
            `;
            elements.resultList.appendChild(item);
        });

        showElement(elements.results);
    }

    function downloadFile(filename, content) {
        const blob = new Blob([content], { type: 'text/calendar;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    async function downloadAllAsZip() {
        if (!splitResults || splitResults.length === 0) return;

        const JSZip = window.JSZip;
        if (!JSZip) {
            for (let i = 0; i < splitResults.length; i++) {
                downloadFile(splitResults[i].filename, splitResults[i].content);
                await new Promise(r => setTimeout(r, 100));
            }
            return;
        }

        const zip = new JSZip();
        splitResults.forEach(function(file) {
            zip.file(file.filename, file.content);
        });

        const blob = await zip.generateAsync({ type: 'blob' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'ical-split-result.zip';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    function initEventListeners() {
        elements.browseBtn.addEventListener('click', function() {
            elements.fileInput.click();
        });

        elements.fileInput.addEventListener('change', function(e) {
            if (e.target.files.length > 0) {
                handleFile(e.target.files[0]);
            }
        });

        elements.dropZone.addEventListener('click', function() {
            elements.fileInput.click();
        });

        elements.dropZone.addEventListener('dragover', function(e) {
            e.preventDefault();
            elements.dropZone.classList.add('drag-over');
        });

        elements.dropZone.addEventListener('dragleave', function() {
            elements.dropZone.classList.remove('drag-over');
        });

        elements.dropZone.addEventListener('drop', function(e) {
            e.preventDefault();
            elements.dropZone.classList.remove('drag-over');
            if (e.dataTransfer.files.length > 0) {
                handleFile(e.dataTransfer.files[0]);
            }
        });

        elements.removeFile.addEventListener('click', resetUI);

        document.querySelectorAll('input[name="split-mode"]').forEach(function(radio) {
            radio.addEventListener('change', function() {
                if (this.value === 'size') {
                    elements.sizeOptions.style.opacity = '1';
                    elements.sizeOptions.style.pointerEvents = 'auto';
                } else {
                    elements.sizeOptions.style.opacity = '0.5';
                    elements.sizeOptions.style.pointerEvents = 'none';
                }
            });
        });

        elements.splitBtn.addEventListener('click', performSplit);

        elements.resultList.addEventListener('click', function(e) {
            if (e.target.classList.contains('download-btn')) {
                const index = parseInt(e.target.dataset.index, 10);
                if (splitResults && splitResults[index]) {
                    downloadFile(splitResults[index].filename, splitResults[index].content);
                }
            }
        });

        elements.downloadAll.addEventListener('click', downloadAllAsZip);
        elements.resetBtn.addEventListener('click', resetUI);
    }

    async function loadJSZip() {
        const script = document.createElement('script');
        script.src = 'https://cdnjs.cloudflare.com/ajax/libs/jszip/3.10.1/jszip.min.js';
        script.integrity = 'sha512-XMVd28F1oH/O71fzwBnV7HucLxVwtxf26XV8P4wPk26EDxuGZ91N8bsOttmnomcCD3CS5ZMRL50H0GgOHvegtg==';
        script.crossOrigin = 'anonymous';
        document.head.appendChild(script);
    }

    async function init() {
        initEventListeners();
        await Promise.all([
            initWasm(),
            loadJSZip(),
        ]);
    }

    init();
})();
