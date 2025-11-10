export namespace diff {
	
	export class DiffLine {
	    type: string;
	    content: string;
	    oldLineNum?: number;
	    newLineNum?: number;
	
	    static createFrom(source: any = {}) {
	        return new DiffLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.content = source["content"];
	        this.oldLineNum = source["oldLineNum"];
	        this.newLineNum = source["newLineNum"];
	    }
	}

}

export namespace main {
	
	export class FileMergeResult {
	    FilePath: string;
	    FileAPath: string;
	    FileBPath: string;
	    OutputPath: string;
	    Changed: number;
	    Added: number;
	    Removed: number;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new FileMergeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FilePath = source["FilePath"];
	        this.FileAPath = source["FileAPath"];
	        this.FileBPath = source["FileBPath"];
	        this.OutputPath = source["OutputPath"];
	        this.Changed = source["Changed"];
	        this.Added = source["Added"];
	        this.Removed = source["Removed"];
	        this.Error = source["Error"];
	    }
	}

}

