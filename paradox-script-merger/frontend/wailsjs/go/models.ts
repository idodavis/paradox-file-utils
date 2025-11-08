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

export namespace patch {
	
	export class ChangedFile {
	    RelativePath: string;
	    FullPathA: string;
	    FullPathB: string;
	    Diff: string;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RelativePath = source["RelativePath"];
	        this.FullPathA = source["FullPathA"];
	        this.FullPathB = source["FullPathB"];
	        this.Diff = source["Diff"];
	        this.Error = source["Error"];
	    }
	}
	export class PatchComparison {
	    ChangedFiles: ChangedFile[];
	    AddedFiles: string[];
	    RemovedFiles: string[];
	
	    static createFrom(source: any = {}) {
	        return new PatchComparison(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ChangedFiles = this.convertValues(source["ChangedFiles"], ChangedFile);
	        this.AddedFiles = source["AddedFiles"];
	        this.RemovedFiles = source["RemovedFiles"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

