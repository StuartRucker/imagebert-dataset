



//torevert: export, el, 
//tokenization takes form of:
// tokenization = [`[UNK]`,`cry`,`##pt`,`##o`,`[UNK]`,`[UNK]`, `economics`,`[UNK]`,`language`,`[UNK]`,`lifestyle`,`[UNK]`,`l`,`##in`,`##ux`,`[UNK]`,`personal`,`[UNK]`,`philosophy`,`[UNK]`,`politics`,`[UNK]`,`religion`,`[UNK]`,`science`,`[UNK]`,`software`,`[UNK]`,`technology`,`[UNK]`,`tradition`,`[UNK]`,`tutor`,`##ial`,`[UNK]`,`updates`]
function maskify(tokenization){

    
    
    function isVisible(el) {
        var rect     = el.getBoundingClientRect(),
            vWidth   = window.innerWidth || doc.documentElement.clientWidth,
            vHeight  = window.innerHeight || doc.documentElement.clientHeight,
            efp      = function (x, y) { return document.elementFromPoint(x, y) };     
    
        // Return false if it's not in the viewport
        if (rect.right < 0 || rect.bottom < 0 
                || rect.left > vWidth || rect.top > vHeight)
            return false;
    
        // Return true if any of its four corners are visible
        return (
              el.contains(efp(rect.left,  rect.top))
          ||  el.contains(efp(rect.right, rect.top))
          ||  el.contains(efp(rect.right, rect.bottom))
          ||  el.contains(efp(rect.left,  rect.bottom))
        );
    }

    function isVisible2(el){
        var elVisible = true
        var observer = new IntersectionObserver(function(entries) {
            if(entries[0]['intersectionRatio'] == 0) {
                elVisible = false
            }
        }, { root: document.documentElement });
        
        // element to observe
        observer.observe(el);
    }
    
    el = this
    try{
        if (! isVisible(el) || ! isVisible2(el)){
            return []
        }
    }catch(err){}

    textNode = el //this
    var text = textNode.nodeValue.toLowerCase()

    
    //iterate through tokens and parir them with the text
    unknown_count = 0
    offsets = []

    current_offset = 0
    for(var i = 0; i < tokenization.length; i++){
        if(tokenization[i] == "[UNK]"){
            unknown_count += 1
            continue
        }

        let use_token = tokenization[i].toLowerCase()
        if(use_token.startsWith("##")){
            use_token = use_token.substring(2)
        }

        let index = text.indexOf(use_token)
        let end_index = index + use_token.length
        let start_index = 0

        if(index == -1){
            unknown_count += 1
            continue
        }

        if(unknown_count > 0){
            unknown_text = text.substring(0, index)
            //partition unknwn text into unknown_count parts
            let unknown_text_parts = []
            let unknown_text_part_length = Math.floor(unknown_text.length / unknown_count)
            let unk_start_index = 0
            for(var j = 0; j < unknown_count; j++){
                let unk_end_index = unk_start_index + unknown_text_part_length
                if(j == unknown_count - 1){
                    unk_end_index = unknown_text.length
                }
                unknown_text_parts.push(unknown_text.substring(unk_start_index, unk_end_index))

                offsets.push( [current_offset + unk_start_index, current_offset + unk_end_index] )

                unk_start_index = unk_end_index
            }
            

            start_index = index
            unknown_count = 0

        }


        offsets.push( [current_offset + start_index, current_offset + end_index] )
        text = text.substring(end_index)

        current_offset += end_index
    }  

    
    //Get a range corresponding to each token
    var final_data = []
    original_text = textNode.nodeValue.toLowerCase()
    let t = textNode.ownerDocument.documentElement.getBoundingClientRect();

    for(var i = 0; i < offsets.length; i++){
        let token = tokenization[i]
        let range = new Range()
        range.setStart(textNode, offsets[i][0])
        range.setEnd(textNode, offsets[i][1])
        
        
        var e = range.getBoundingClientRect()
        
        
        new_data = {
            x: e.left - t.left,
            y: e.top - t.top,
            width: e.width,
            height: e.height,
            token: token,
            word: original_text.substring(offsets[i][0], offsets[i][1])
        }
        if (new_data.width > 5 && new_data.height > 5) {
            final_data.push(new_data)
        }

    }
    return {"data":final_data}

}   