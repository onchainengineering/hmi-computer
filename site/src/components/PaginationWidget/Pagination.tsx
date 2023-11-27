import {
  type FC,
  type HTMLAttributes,
  useEffect,
  useLayoutEffect,
  useRef,
} from "react";

import { useEffectEvent } from "hooks/hookPolyfills";
import { PaginationWidgetBase } from "./PaginationWidgetBase";

type PaginationProps = HTMLAttributes<HTMLDivElement> & {
  currentPage: number;
  pageSize: number;
  totalRecords: number | undefined;
  onPageChange: (newPage: number) => void;
  autoScroll?: boolean;

  /**
   * Meant to interface with useQuery's isPreviousData property.
   *
   * Indicates whether data for a previous query is being shown while a new
   * query is loading in
   */
  showingPreviousData: boolean;
};

const userInteractionEvents: (keyof WindowEventMap)[] = [
  "click",
  "scroll",
  "pointerenter",
  "touchstart",
  "keydown",
];

export const Pagination: FC<PaginationProps> = ({
  children,
  currentPage,
  pageSize,
  totalRecords,
  showingPreviousData,
  onPageChange,
  autoScroll = true,
  ...delegatedProps
}) => {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const isScrollingQueuedRef = useRef(false);

  useEffect(() => {
    const cancelScroll = () => {
      isScrollingQueuedRef.current = false;
    };

    for (const event of userInteractionEvents) {
      window.addEventListener(event, cancelScroll);
    }

    return () => {
      for (const event of userInteractionEvents) {
        window.removeEventListener(event, cancelScroll);
      }
    };
  }, []);

  /**
   * This function is mainly accounting for five different triggers for the
   * below useLayoutEffect call:
   *
   * 1. Initial render – We don't want anything to run on the initial render to
   *    avoid hijacking the user's browser and also make sure the UI doesn't
   *    feel janky. showingPreviousData should always be false, and currentPage
   *    should generally be 1.
   * 2. Current page doesn’t change, but showingPreviousData becomes true - Also
   *    do nothing.
   * 3. Current page doesn’t change, but showingPreviousData becomes false - The
   *    data for the current page has finally come in; scroll if a scroll is
   *    queued from a previous render
   * 4. Current page changes and showingPreviousData is false – we have cached
   *    data for whatever page we just jumped to. Scroll immediately (and reset
   *    the scrolling state just to be on the safe side)
   * 5. Current page changes and showingPreviousData is false – Cache miss.
   *    Queue up a scroll, but do nothing else. If the user does anything at all
   *    while the new data is loading in, cancel the scroll.
   *
   * Set up as an effect event because currentPage and showingPreviousData
   * should be the only two cues for syncing scroll position
   */
  const syncScrollChange = useEffectEvent(() => {
    if (showingPreviousData) {
      isScrollingQueuedRef.current = true;
      return;
    }

    if (autoScroll && isScrollingQueuedRef.current) {
      scrollContainerRef.current?.scrollIntoView({
        block: "start",
        behavior: "instant",
      });
    }

    isScrollingQueuedRef.current = false;
  });

  useLayoutEffect(() => {
    syncScrollChange();
  }, [syncScrollChange, currentPage, showingPreviousData]);

  return (
    <div ref={scrollContainerRef}>
      <div
        css={{
          display: "flex",
          flexFlow: "column nowrap",
          rowGap: "24px",
        }}
        {...delegatedProps}
      >
        {children}
        {totalRecords !== undefined && (
          <PaginationWidgetBase
            currentPage={currentPage}
            pageSize={pageSize}
            totalRecords={totalRecords}
            onPageChange={onPageChange}
          />
        )}
      </div>
    </div>
  );
};
