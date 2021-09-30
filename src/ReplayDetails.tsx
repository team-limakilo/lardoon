import React, { useMemo, useState } from "react";
import { BiDownload } from "react-icons/bi";
import { Link } from "react-router-dom";
import useFetch, { CachePolicies } from "use-http";
import { searchStore } from "./App";

export type ReplayObject = {
  id: number;
  replay_id: number;
  name: string;
  pilot: string;
  created_offset: number;
  deleted_offset: number;
};

export type Replay = {
  id: number;
  path: string;
  reference_time: string;
  recording_time: string;
  title: string;
  data_source: string;
  data_recorder: string;
  duration: number;
  size: number;
};

type DownloadRequest = {
  object_ids?: Array<number>;
  before?: number;
  after?: number;
};

function ReplayObject(
  { object, active, setActive }: {
    object: ReplayObject;
    active: boolean;
    setActive: () => void;
  },
) {
  const startTime = new Date(0);
  const endTime = new Date(0);
  startTime.setSeconds(object.created_offset);
  endTime.setSeconds(object.deleted_offset);
  const duration = new Date(0);
  duration.setSeconds(object.deleted_offset - object.created_offset);

  return (
    <div
      className="border border-gray-200 rounded-sm cursor-pointer"
      onClick={setActive}
    >
      <div
        className="flex flex-row items-center bg-gray-300 p-2"
      >
        <div>{object.pilot} ({object.name})</div>
        <div className="flex flex-row ml-auto gap-4 items-center">
          <div className="text-gray-600 text-sm">
            {startTime.toISOString().substr(11, 8)}
            {" - "}
            {endTime.toISOString().substr(11, 8)}
          </div>
        </div>
      </div>
      {active && (
        <div className="bg-gray-50 p-2 flex flex-col">
          <div className="flex flex-row">
            <b className="text-bold mr-2">Duration:</b>
            {duration.toISOString().substr(11, 8)}
          </div>
          <div className="flex flex-row">
            <a
              className="p-2 border border-green-500 bg-green-200 hover:bg-green-300 text-green-700 rounded-sm shadow-sm text-sm ml-auto"
              href={`/api/replay/${object.replay_id}/download?start=${object
                .created_offset - 10}&end=${object.deleted_offset + 10}`}
              onClick={(e) => e.stopPropagation()}
            >
              Download
              <BiDownload className="inline-flex w-4 h-4 ml-1" />
            </a>
          </div>
        </div>
      )}
    </div>
  );
}

export default function ReplayDetails({ replayId }: { replayId: number }) {
  const { data } = useFetch<Replay & { objects: Array<ReplayObject> }>(
    `/api/replay/${replayId}`,
    {
      cachePolicy: CachePolicies.NO_CACHE,
    },
    [],
  );

  const [search, setSearch] = searchStore(
    (state) => [state.value, state.setValue],
  );
  const [objectId, setObjectId] = useState<number | null>(null);

  const selectedObjects = useMemo(() => {
    if (!data) {
      return null;
    }

    return data.objects.sort((a, b) => a.created_offset - b.created_offset)
      .filter((it) =>
        search === "" ||
        (it.name.toLowerCase().includes(search.toLowerCase()) ||
          it.pilot.toLowerCase().includes(search.toLowerCase()))
      );
  }, [data, search]);

  if (!selectedObjects) return <></>;

  return (
    <div className="flex flex-col h-full mx-auto md:w-1/3 w-full">
      <div className="p-2 flex flex-row gap-2">
        <input
          className="border border-gray-300 rounded-sm form-input w-full h-8 p-0.5"
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Link
          to="/"
          className="p-1 text-gray-700 bg-gray-200 border-gray-400 rounded-sm hover:text-blue-500"
        >
          Back
        </Link>
      </div>
      <div className="flex flex-col gap-2 p-2">
        {selectedObjects.map((it) => (
          <ReplayObject
            key={it.id}
            object={it}
            active={objectId === it.id}
            setActive={() =>
              objectId === it.id ? setObjectId(null) : setObjectId(it.id)}
          />
        ))}
      </div>
    </div>
  );
}
